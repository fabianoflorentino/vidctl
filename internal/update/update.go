// Package update checks GitHub for a newer vidctl release. It talks to the
// unauthenticated releases API with a short timeout and caches the outcome for
// the session, so offline or rate-limited machines degrade to a silent
// "no update" instead of bothering the user.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultEndpoint is the GitHub releases endpoint the production checker
	// queries. The most recent release is per platform (v2.5.0-linux is the
	// latest), so the base tag is recovered by stripping the platform suffix.
	DefaultEndpoint = "https://api.github.com/repos/fabianoflorentino/vidctl/releases/latest"
	requestTimeout  = 5 * time.Second
)

// Info is the outcome of a version check. Ok is false whenever the network or
// the API is unavailable — callers treat that as a silent skip, never as an
// error the user must see.
type Info struct {
	Ok        bool   `json:"ok"`
	HasUpdate bool   `json:"hasUpdate"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	URL       string `json:"url"`
	AssetURL  string `json:"assetURL"`
}

// Checker queries the releases endpoint with a per-session cache. Only
// successful checks are cached, so a manual re-check after a network failure
// actually retries.
type Checker struct {
	endpoint string
	client   *http.Client
	platform string

	mu     sync.Mutex
	cached *Info
}

// New returns a Checker for the given platform ("linux", "windows" or
// "macos"), which decides which release asset URL is reported.
func New(platform string) *Checker {
	return NewWithEndpoint(DefaultEndpoint, platform)
}

// NewWithEndpoint returns a Checker pinned to a custom endpoint and platform.
// Tests inject an httptest server this way.
func NewWithEndpoint(endpoint, platform string) *Checker {
	return &Checker{
		endpoint: endpoint,
		client:   &http.Client{Timeout: requestTimeout},
		platform: platform,
	}
}

// Check reports whether a newer release exists than current. Delivery errors
// yield Info{Ok: false} with a nil error: the check must never block a task.
// The fetch runs under the lock, so concurrent callers share a single request
// and the session cache.
func (c *Checker) Check(current string) (Info, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cached != nil {
		return *c.cached, nil
	}

	latest, err := c.fetch()
	if err != nil {
		return Info{Ok: false}, nil
	}
	info := evaluate(latest, current, c.platform)
	info.Ok = true

	c.cached = &info
	return info, nil
}

type release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func (c *Checker) fetch() (release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "vidctl/update-check")

	resp, err := c.client.Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("releases API replied with status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return release{}, err
	}
	var rel release
	if err := json.Unmarshal(body, &rel); err != nil {
		return release{}, err
	}
	return rel, nil
}

// evaluate builds the Info for a release payload. A tag that cannot be parsed,
// a "dev" local build or a missing asset yields HasUpdate=false.
func evaluate(rel release, current, platform string) Info {
	info := Info{Current: current, Latest: baseTag(rel.TagName), URL: rel.HTMLURL}
	if !isNewer(rel.TagName, current) {
		return info
	}
	info.HasUpdate = true
	info.AssetURL = pickAsset(rel.Assets, platform)
	return info
}

// pickAsset returns the download URL of the zip for the given platform, or ""
// when the release has no matching asset.
func pickAsset(assets []struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}, platform string) string {
	prefix := "vidctl-" + platform + "-"
	for _, a := range assets {
		if strings.HasPrefix(a.Name, prefix) && strings.HasSuffix(a.Name, ".zip") {
			return a.URL
		}
	}
	return ""
}

// baseTag strips the platform suffix of a per-OS tag (v2.5.0-linux → v2.5.0).
func baseTag(tag string) string {
	if i := strings.LastIndex(tag, "-"); i > 0 && !strings.ContainsAny(tag[i+1:], "0123456789") {
		return tag[:i]
	}
	return tag
}

// isNewer reports whether tag is a newer semver than current. Unparseable
// values (local "dev" builds, tags without a version) mean "no update".
func isNewer(tag, current string) bool {
	cur, ok := semverParts(baseTag(tag))
	if !ok {
		return false
	}
	base, ok := semverParts(current)
	if !ok {
		return false
	}
	for i := range cur {
		if cur[i] > base[i] {
			return true
		}
		if cur[i] < base[i] {
			return false
		}
	}
	return false
}

func semverParts(v string) ([3]int, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}
