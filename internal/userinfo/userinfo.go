package userinfo

import (
	"net/http"
	"strconv"
	"strings"
)

// Stats holds traffic quota fields from the Subscription-Userinfo header.
type Stats struct {
	Upload   uint64
	Download uint64
	Total    uint64
	Expire   int64 // Unix seconds; 0 means unknown / not provided
}

// Used returns upload + download.
func (s Stats) Used() uint64 {
	return s.Upload + s.Download
}

// ParseFromResponse extracts stats from response headers (name is case-insensitive).
func ParseFromResponse(resp *http.Response) (Stats, bool) {
	if resp == nil {
		return Stats{}, false
	}
	raw := ""
	for k, v := range resp.Header {
		if strings.EqualFold(k, "Subscription-Userinfo") && len(v) > 0 {
			raw = v[0]
			break
		}
	}
	if raw == "" {
		return Stats{}, false
	}
	return ParseHeaderValue(raw)
}

// ParseHeaderValue parses the semicolon-separated key=value form.
func ParseHeaderValue(value string) (Stats, bool) {
	var st Stats
	st.Expire = -1
	any := false
	parts := strings.Split(value, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])
		switch key {
		case "upload", "download", "total":
			n, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				continue
			}
			any = true
			switch key {
			case "upload":
				st.Upload = n
			case "download":
				st.Download = n
			case "total":
				st.Total = n
			}
		case "expire":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				continue
			}
			any = true
			st.Expire = n
		}
	}
	if st.Expire < 0 {
		st.Expire = 0
	}
	if !any {
		return Stats{}, false
	}
	return st, true
}
