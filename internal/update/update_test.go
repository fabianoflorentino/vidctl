package update

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func releaseBody(tag string) string {
	return `{
	  "tag_name": "` + tag + `",
	  "html_url": "https://github.com/fabianoflorentino/vidctl/releases/tag/` + tag + `",
	  "assets": [
	    {"name": "vidctl-linux-x64.zip", "browser_download_url": "https://dl/vidctl-linux-x64.zip"},
	    {"name": "vidctl-windows-x64.zip", "browser_download_url": "https://dl/vidctl-windows-x64.zip"},
	    {"name": "vidctl-macos-arm64.zip", "browser_download_url": "https://dl/vidctl-macos-arm64.zip"},
	    {"name": "checksums.txt", "browser_download_url": "https://dl/checksums.txt"}
	  ]
	}`
}

func TestCheckFindsNewerRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(releaseBody("v2.6.0-linux")))
	}))
	defer srv.Close()

	c := NewWithEndpoint(srv.URL, "linux")
	info, err := c.Check("v2.5.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !info.Ok || !info.HasUpdate {
		t.Errorf("info = %+v; want ok+hasUpdate", info)
	}
	if info.Latest != "v2.6.0" {
		t.Errorf("Latest = %q; want v2.6.0", info.Latest)
	}
	if info.AssetURL != "https://dl/vidctl-linux-x64.zip" {
		t.Errorf("AssetURL = %q; want linux zip", info.AssetURL)
	}
	if info.URL == "" {
		t.Error("URL should point to the release page")
	}
}

func TestCheckNoUpdateWhenCurrentIsLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(releaseBody("v2.5.0-linux")))
	}))
	defer srv.Close()

	info, err := NewWithEndpoint(srv.URL, "windows").Check("v2.5.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !info.Ok || info.HasUpdate {
		t.Errorf("info = %+v; want ok without update", info)
	}
}

func TestCheckDevBuildNeverReportsUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(releaseBody("v2.6.0-linux")))
	}))
	defer srv.Close()

	info, err := NewWithEndpoint(srv.URL, "linux").Check("dev")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.HasUpdate {
		t.Errorf("info = %+v; a dev build must not claim an update", info)
	}
}

func TestCheckHTTPErrorIsSilent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	info, err := NewWithEndpoint(srv.URL, "linux").Check("v2.5.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Ok || info.HasUpdate {
		t.Errorf("info = %+v; want silent no-check", info)
	}
}

func TestCheckNotFoundIsSilent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	info, err := NewWithEndpoint(srv.URL, "linux").Check("v2.5.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Ok {
		t.Errorf("info = %+v; a 404 must be silent", info)
	}
}

func TestCheckNetworkErrorIsSilent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(releaseBody("v2.6.0-linux")))
	}))
	url := srv.URL
	srv.Close()

	info, err := NewWithEndpoint(url, "linux").Check("v2.5.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Ok {
		t.Errorf("info = %+v; a closed server must be silent", info)
	}
}

func TestCheckCachesPerSession(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(releaseBody("v2.6.0-linux")))
	}))
	defer srv.Close()

	c := NewWithEndpoint(srv.URL, "linux")
	if _, err := c.Check("v2.5.0"); err != nil {
		t.Fatalf("first Check: %v", err)
	}
	if _, err := c.Check("v2.5.0"); err != nil {
		t.Fatalf("second Check: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("endpoint called %d times; want 1 (session cache)", got)
	}
}

func TestCheckRetriesAfterFailure(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(releaseBody("v2.6.0-linux")))
	}))
	defer srv.Close()

	c := NewWithEndpoint(srv.URL, "linux")
	if info, _ := c.Check("v2.5.0"); info.Ok {
		t.Error("first check should be silent (500)")
	}
	info, err := c.Check("v2.5.0")
	if err != nil {
		t.Fatalf("second Check: %v", err)
	}
	if !info.Ok || !info.HasUpdate {
		t.Errorf("info = %+v; want ok+hasUpdate after retry", info)
	}
}

func TestPickAssetPerPlatform(t *testing.T) {
	cases := []struct {
		platform, want string
	}{
		{"linux", "https://dl/vidctl-linux-x64.zip"},
		{"windows", "https://dl/vidctl-windows-x64.zip"},
		{"macos", "https://dl/vidctl-macos-arm64.zip"},
		{"plan9", ""},
	}
	for _, tt := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(releaseBody("v2.6.0-linux")))
		}))
		info, err := NewWithEndpoint(srv.URL, tt.platform).Check("v2.5.0")
		srv.Close()
		if err != nil {
			t.Fatalf("Check(%s): %v", tt.platform, err)
		}
		if info.AssetURL != tt.want {
			t.Errorf("platform %q: AssetURL = %q; want %q", tt.platform, info.AssetURL, tt.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		tag, current string
		want         bool
	}{
		{"v2.6.0-linux", "v2.5.0", true},
		{"v2.5.0-linux", "v2.5.0", false},
		{"v2.5.1-linux", "v2.5.0", true},
		{"v2.5.0-linux", "v2.4.9", true},
		{"v2.5.0-linux", "v3.0.0", false},
		{"v2.5.0", "", false},
		{"v2.5.0-linux", "dev", false},
		{"not-a-version", "v2.5.0", false},
	}
	for _, tt := range cases {
		if got := isNewer(tt.tag, tt.current); got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v; want %v", tt.tag, tt.current, got, tt.want)
		}
	}
}

func TestConcurrentChecksCacheOnce(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond)
		w.Write([]byte(releaseBody("v2.6.0-linux")))
	}))
	defer srv.Close()

	c := NewWithEndpoint(srv.URL, "linux")
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			info, _ := c.Check("v2.5.0")
			if !info.Ok {
				t.Error("concurrent check should be ok")
			}
		}()
	}
	wg.Wait()
	if got := calls.Load(); got > 1 {
		t.Errorf("endpoint called %d times; want at most 1 (concurrent cache)", got)
	}
}
