package anyshare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
