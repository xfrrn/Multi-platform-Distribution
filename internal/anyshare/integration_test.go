package anyshare

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRealAnyshareUploadDownload(t *testing.T) {
	if os.Getenv("ANYSHARE_REAL_TEST") != "1" {
		t.Skip("set ANYSHARE_REAL_TEST=1 to run against a real Anyshare share")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client, err := NewClient(ctx, Config{
		BaseURL:         os.Getenv("ANYSHARE_BASE_URL"),
		SharingLink:     os.Getenv("ANYSHARE_SHARING_LINK"),
		UploadPath:      os.Getenv("ANYSHARE_UPLOAD_PATH"),
		Cookie:          os.Getenv("ANYSHARE_COOKIE"),
		Timeout:         30 * time.Second,
		RefreshInterval: envDuration("ANYSHARE_REFRESH_INTERVAL", 30*time.Minute),
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	name := "mpd-go-client-test-" + time.Now().Format("20060102-150405") + ".txt"
	content := "mpd go client anyshare test " + time.Now().Format(time.RFC3339Nano)
	uploaded, err := client.Upload(ctx, name, strings.NewReader(content))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	resolved, err := client.ResolveDownload(ctx, uploaded.DocID, uploaded.Rev, uploaded.Name)
	if err != nil {
		t.Fatalf("resolve download: %v", err)
	}
	resp, err := http.Get(resolved.URL)
	if err != nil {
		t.Fatalf("download direct url: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected download status 200, got %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read download body: %v", err)
	}
	if string(data) != content {
		t.Fatalf("downloaded content mismatch for %s", uploaded.DocID)
	}
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}
