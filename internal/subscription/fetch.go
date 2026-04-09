package subscription

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"v2ray-subscription-data-usage-monitor/internal/userinfo"
)

const userAgent = "v2ray-subscription-data-usage-monitor/1.0"

// Result is the outcome of a subscription probe.
type Result struct {
	Stats       userinfo.Stats
	HasUserinfo bool
	StatusCode  int
	BodySnippet string
	Err         error
}

// Fetch performs a GET to the subscription URL and reads userinfo from headers.
func Fetch(url string, client *http.Client) Result {
	url = strings.TrimSpace(url)
	var r Result
	if url == "" {
		r.Err = fmt.Errorf("subscription URL is empty")
		return r
	}
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		r.Err = err
		return r
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		r.Err = err
		return r
	}
	defer resp.Body.Close()

	r.StatusCode = resp.StatusCode
	const maxBody = 512
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	r.BodySnippet = strings.TrimSpace(string(snippet))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		r.Err = fmt.Errorf("HTTP %d", resp.StatusCode)
		return r
	}

	st, ok := userinfo.ParseFromResponse(resp)
	r.Stats = st
	r.HasUserinfo = ok
	if !ok {
		r.Err = fmt.Errorf("no Subscription-Userinfo header in response")
	}
	return r
}
