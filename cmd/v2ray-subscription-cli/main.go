// Command v2ray-subscription-cli is a pure-Go, no-GUI probe: GET a subscription URL and print
// usage from the Subscription-Userinfo header. Binary size is typically a few MB (no Fyne/GLFW).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"v2ray-subscription-data-usage-monitor/internal/subscription"
)

func main() {
	url := flag.String("url", "", "subscription URL (GET)")
	timeout := flag.Int("timeout", 45, "HTTP client timeout in seconds")
	asJSON := flag.Bool("json", false, "print one JSON object on stdout")
	flag.Parse()

	if *url == "" {
		fmt.Fprintln(os.Stderr, "usage: v2ray-subscription-cli -url <subscription-url> [-timeout N] [-json]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	if *timeout < 1 {
		*timeout = 45
	}
	client := &http.Client{Timeout: time.Duration(*timeout) * time.Second}
	res := subscription.Fetch(*url, client)

	if res.Err != nil {
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
				"ok":          false,
				"error":       res.Err.Error(),
				"status_code": res.StatusCode,
			})
		} else {
			if res.StatusCode != 0 {
				fmt.Fprintf(os.Stderr, "error: %v (HTTP %d)\n", res.Err, res.StatusCode)
			} else {
				fmt.Fprintf(os.Stderr, "error: %v\n", res.Err)
			}
		}
		os.Exit(1)
	}

	st := res.Stats
	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":           true,
			"upload":       st.Upload,
			"download":     st.Download,
			"used":         st.Used(),
			"total":        st.Total,
			"expire_unix":  st.Expire,
			"status_code":  res.StatusCode,
		})
		return
	}

	fmt.Printf("upload=%d download=%d used=%d total=%d expire=%d\n",
		st.Upload, st.Download, st.Used(), st.Total, st.Expire)
}
