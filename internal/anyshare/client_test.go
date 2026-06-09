package anyshare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientUploadAndResolveDownload(t *testing.T) {
	const sharingID = "AA121158B8D88B4E7C9019EA24FD02E541"
	var sawBegin, sawUpload, sawEnd, sawDownload bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/link/" + sharingID:
			w.Header().Add("Set-Cookie", "link_token:"+sharingID+"=anon-token; Path=/")
			w.WriteHeader(http.StatusOK)
		case "/api/efast/v1/file/osbeginupload":
			sawBegin = true
			if r.Header.Get("Authorization") != "Bearer anon-token" {
				t.Fatalf("expected bearer token on beginupload, got %q", r.Header.Get("Authorization"))
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode beginupload body: %v", err)
			}
			if body["docid"] != "gns://upload-dir" || body["name"] != "installer.exe" {
				t.Fatalf("unexpected beginupload body: %#v", body)
			}
			writeJSON(w, map[string]any{
				"docid": "gns://file-docid",
				"rev":   "file-rev",
				"authrequest": []string{
					"POST",
					serverURL(r) + "/upload",
					"AWSAccessKeyId: key",
					"Content-Type: application/octet-stream",
					"Policy: policy",
					"Signature: signature",
					"key: object-key",
				},
			})
		case "/upload":
			sawUpload = true
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse multipart upload: %v", err)
			}
			if r.FormValue("key") != "object-key" {
				t.Fatalf("expected upload key field, got %q", r.FormValue("key"))
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("read uploaded file: %v", err)
			}
			_ = file.Close()
			if header.Filename != "installer.exe" {
				t.Fatalf("expected uploaded filename installer.exe, got %s", header.Filename)
			}
			w.WriteHeader(http.StatusNoContent)
		case "/api/efast/v1/file/osendupload":
			sawEnd = true
			if r.Header.Get("Authorization") != "Bearer anon-token" {
				t.Fatalf("expected bearer token on endupload, got %q", r.Header.Get("Authorization"))
			}
			writeJSON(w, map[string]any{"ok": true})
		case "/api/efast/v1/file/osdownload":
			sawDownload = true
			if r.Header.Get("Authorization") != "Bearer anon-token" {
				t.Fatalf("expected bearer token on osdownload, got %q", r.Header.Get("Authorization"))
			}
			writeJSON(w, map[string]any{
				"size":        7,
				"authrequest": []string{"GET", serverURL(r) + "/direct/installer.exe?token=download"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), Config{
		BaseURL:     server.URL,
		SharingLink: server.URL + "/link/" + sharingID,
		UploadPath:  "gns://upload-dir",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	uploaded, err := client.Upload(context.Background(), "installer.exe", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if uploaded.DocID != "gns://file-docid" || uploaded.Rev != "file-rev" || uploaded.Size != 7 || uploaded.SHA512 == "" {
		t.Fatalf("unexpected upload result: %#v", uploaded)
	}

	download, err := client.ResolveDownload(context.Background(), uploaded.DocID, uploaded.Rev, uploaded.Name)
	if err != nil {
		t.Fatalf("resolve download: %v", err)
	}
	if !strings.Contains(download.URL, "/direct/installer.exe?token=download") {
		t.Fatalf("unexpected download url %s", download.URL)
	}
	if !sawBegin || !sawUpload || !sawEnd || !sawDownload {
		t.Fatalf("expected all anyshare calls, begin=%v upload=%v end=%v download=%v", sawBegin, sawUpload, sawEnd, sawDownload)
	}
}

func TestClientUploadRefreshesAuthorizationFromCookie(t *testing.T) {
	const sharingID = "AA60DDB0BEB3F141A98A7DF75B5F5D7992"
	var sawRefresh bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/anyshare/oauth2/login/refreshToken":
			sawRefresh = true
			if r.URL.Query().Get("force") != "false" {
				t.Fatalf("expected non-forced initial refresh, got %q", r.URL.RawQuery)
			}
			w.Header().Add("Set-Cookie", "Authorization=Bearer login-token; Path=/")
			writeJSON(w, map[string]any{"ok": true})
		case "/api/efast/v1/file/osbeginupload":
			if r.Header.Get("Authorization") != "Bearer login-token" {
				t.Fatalf("expected refreshed authorization on beginupload, got %q", r.Header.Get("Authorization"))
			}
			writeJSON(w, map[string]any{
				"docid": "gns://file-docid",
				"rev":   "file-rev",
				"authrequest": []string{
					"POST",
					serverURL(r) + "/upload",
					"AWSAccessKeyId: key",
					"Content-Type: application/octet-stream",
					"Policy: policy",
					"Signature: signature",
					"key: object-key",
				},
			})
		case "/upload":
			w.WriteHeader(http.StatusNoContent)
		case "/api/efast/v1/file/osendupload":
			if r.Header.Get("Authorization") != "Bearer login-token" {
				t.Fatalf("expected refreshed authorization on endupload, got %q", r.Header.Get("Authorization"))
			}
			writeJSON(w, map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), Config{
		BaseURL:     server.URL,
		SharingLink: server.URL + "/anyshare/zh-cn/link/" + sharingID,
		UploadPath:  "gns://upload-dir",
		Cookie:      "SESSION=logged-in",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = client.Upload(context.Background(), "installer.exe", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if !sawRefresh {
		t.Fatal("expected refresh token endpoint to be called")
	}
}

func TestClientRetriesObjectUploadAfterDisconnect(t *testing.T) {
	const sharingID = "AA60DDB0BEB3F141A98A7DF75B5F5D7992"
	var uploadCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/link/" + sharingID:
			w.Header().Add("Set-Cookie", "link_token:"+sharingID+"=anon-token; Path=/")
			w.WriteHeader(http.StatusOK)
		case "/api/efast/v1/file/osbeginupload":
			writeJSON(w, map[string]any{
				"docid": "gns://file-docid",
				"rev":   "file-rev",
				"authrequest": []string{
					"POST",
					serverURL(r) + "/upload",
					"AWSAccessKeyId: key",
					"Content-Type: application/octet-stream",
					"Policy: policy",
					"Signature: signature",
					"key: object-key",
				},
			})
		case "/upload":
			count := atomic.AddInt32(&uploadCalls, 1)
			if count == 1 {
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Fatalf("hijack upload connection: %v", err)
				}
				_ = conn.Close()
				return
			}
			if len(r.TransferEncoding) != 0 {
				t.Fatalf("expected upload to use content-length, got transfer encoding %#v", r.TransferEncoding)
			}
			if r.ContentLength <= int64(len("payload")) {
				t.Fatalf("expected multipart upload content-length, got %d", r.ContentLength)
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse retried multipart upload: %v", err)
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("read retried uploaded file: %v", err)
			}
			_ = file.Close()
			if header.Filename != "installer.exe" {
				t.Fatalf("expected retried filename installer.exe, got %s", header.Filename)
			}
			w.WriteHeader(http.StatusNoContent)
		case "/api/efast/v1/file/osendupload":
			writeJSON(w, map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), Config{
		BaseURL:     server.URL,
		SharingLink: server.URL + "/link/" + sharingID,
		UploadPath:  "gns://upload-dir",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = client.Upload(context.Background(), "installer.exe", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("upload after retry: %v", err)
	}
	if uploadCalls != 2 {
		t.Fatalf("expected object upload to be retried once, got %d calls", uploadCalls)
	}
}

func TestClientRetriesPostJSONAfterUnauthorizedRefresh(t *testing.T) {
	const sharingID = "AA60DDB0BEB3F141A98A7DF75B5F5D7992"
	var beginCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/anyshare/oauth2/login/refreshToken":
			if r.URL.Query().Get("force") == "true" {
				w.Header().Add("Set-Cookie", "Authorization=Bearer fresh-token; Path=/")
			} else {
				w.Header().Add("Set-Cookie", "Authorization=Bearer stale-token; Path=/")
			}
			writeJSON(w, map[string]any{"ok": true})
		case "/api/efast/v1/file/osdownload":
			beginCalls++
			if beginCalls == 1 {
				if r.Header.Get("Authorization") != "Bearer stale-token" {
					t.Fatalf("expected stale authorization before retry, got %q", r.Header.Get("Authorization"))
				}
				http.Error(w, "expired", http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer fresh-token" {
				t.Fatalf("expected fresh authorization after retry, got %q", r.Header.Get("Authorization"))
			}
			writeJSON(w, map[string]any{
				"size":        7,
				"authrequest": []string{"GET", serverURL(r) + "/direct/installer.exe?token=download"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), Config{
		BaseURL:     server.URL,
		SharingLink: server.URL + "/link/" + sharingID,
		UploadPath:  "gns://upload-dir",
		Cookie:      "SESSION=logged-in",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	download, err := client.ResolveDownload(context.Background(), "gns://file-docid", "file-rev", "installer.exe")
	if err != nil {
		t.Fatalf("resolve download: %v", err)
	}
	if !strings.Contains(download.URL, "/direct/installer.exe?token=download") {
		t.Fatalf("unexpected download url %s", download.URL)
	}
	if beginCalls != 2 {
		t.Fatalf("expected download request to be retried once, got %d calls", beginCalls)
	}
}

func TestClientRefreshesAuthorizationOnInterval(t *testing.T) {
	const sharingID = "AA60DDB0BEB3F141A98A7DF75B5F5D7992"
	var refreshCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/anyshare/oauth2/login/refreshToken":
			count := atomic.AddInt32(&refreshCalls, 1)
			w.Header().Add("Set-Cookie", fmt.Sprintf("Authorization=Bearer token-%d; Path=/", count))
			writeJSON(w, map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), Config{
		BaseURL:         server.URL,
		SharingLink:     server.URL + "/link/" + sharingID,
		UploadPath:      "gns://upload-dir",
		Cookie:          "SESSION=logged-in",
		Timeout:         time.Second,
		RefreshInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	defer client.Close()

	deadline := time.After(500 * time.Millisecond)
	for atomic.LoadInt32(&refreshCalls) < 2 {
		select {
		case <-deadline:
			t.Fatalf("expected interval refresh after initial call, got %d refreshes", atomic.LoadInt32(&refreshCalls))
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestClientRetriesPostJSONAfterUnauthorizedAnonymous(t *testing.T) {
	const sharingID = "AA121158B8D88B4E7C9019EA24FD02E541"
	var linkVisits, beginCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/link/" + sharingID:
			linkVisits++
			if linkVisits == 1 {
				w.Header().Add("Set-Cookie", "link_token:"+sharingID+"=stale-anon-token; Path=/")
			} else {
				w.Header().Add("Set-Cookie", "link_token:"+sharingID+"=fresh-anon-token; Path=/")
			}
			w.WriteHeader(http.StatusOK)
		case "/api/efast/v1/file/osbeginupload":
			beginCalls++
			if beginCalls == 1 {
				if r.Header.Get("Authorization") != "Bearer stale-anon-token" {
					t.Fatalf("expected stale anon token on first attempt, got %q", r.Header.Get("Authorization"))
				}
				http.Error(w, "token expired", http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer fresh-anon-token" {
				t.Fatalf("expected fresh anon token on retry, got %q", r.Header.Get("Authorization"))
			}
			writeJSON(w, map[string]any{
				"docid": "gns://file-docid",
				"rev":   "file-rev",
				"authrequest": []string{
					"POST",
					serverURL(r) + "/upload",
					"AWSAccessKeyId: key",
					"Content-Type: application/octet-stream",
					"Policy: policy",
					"Signature: signature",
					"key: object-key",
				},
			})
		case "/upload":
			w.WriteHeader(http.StatusNoContent)
		case "/api/efast/v1/file/osendupload":
			if r.Header.Get("Authorization") != "Bearer fresh-anon-token" {
				t.Fatalf("expected fresh anon token on endupload, got %q", r.Header.Get("Authorization"))
			}
			writeJSON(w, map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), Config{
		BaseURL:     server.URL,
		SharingLink: server.URL + "/link/" + sharingID,
		UploadPath:  "gns://upload-dir",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = client.Upload(context.Background(), "installer.exe", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if linkVisits != 2 {
		t.Fatalf("expected sharing link visited twice (initial + retry), got %d", linkVisits)
	}
	if beginCalls != 2 {
		t.Fatalf("expected beginupload called twice, got %d", beginCalls)
	}
}

func TestClientRejectsUnexpectedDownloadAuth(t *testing.T) {
	_, err := parseDownloadAuth([]string{"POST", "https://example.test/file"})
	if err == nil {
		t.Fatal("expected invalid download method to fail")
	}
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}
