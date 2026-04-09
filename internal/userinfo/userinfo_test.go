package userinfo

import (
	"net/http"
	"testing"
)

func TestParseHeaderValue(t *testing.T) {
	st, ok := ParseHeaderValue("upload=100; download=200; total=1000; expire=1700000000")
	if !ok {
		t.Fatal("expected ok")
	}
	if st.Upload != 100 || st.Download != 200 || st.Total != 1000 || st.Expire != 1700000000 {
		t.Fatalf("unexpected: %+v", st)
	}
	if st.Used() != 300 {
		t.Fatalf("used: %d", st.Used())
	}
}

func TestParseFromResponse_CaseInsensitiveHeader(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"Subscription-Userinfo": {"upload=1; download=2; total=10"},
		},
	}
	st, ok := ParseFromResponse(resp)
	if !ok {
		t.Fatal("expected ok")
	}
	if st.Used() != 3 || st.Total != 10 {
		t.Fatalf("unexpected: %+v", st)
	}
}
