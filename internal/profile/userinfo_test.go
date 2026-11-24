package profile

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Fetch reads the plan's traffic from the subscription-userinfo header, as
// panels such as 3x-ui send it.
func TestFetchReadsUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("subscription-userinfo", "upload=1048576; download=3145728; total=107374182400; expire=1800000000")
		w.Header().Set("profile-update-interval", "12")
		_, _ = w.Write([]byte(`proxies:
  - {name: a, type: socks5, server: 127.0.0.1, port: 1080}
rules:
  - MATCH,DIRECT
`))
	}))
	defer server.Close()

	_, info, err := Fetch(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Upload != 1048576 || info.Download != 3145728 {
		t.Errorf("已用 upload=%d download=%d", info.Upload, info.Download)
	}
	if info.Total != 107374182400 {
		t.Errorf("月总量 %d", info.Total)
	}
	if info.Interval != 12 {
		t.Errorf("更新间隔 %d", info.Interval)
	}
	if !info.Expire.Equal(time.Unix(1800000000, 0)) {
		t.Errorf("到期时间 %v", info.Expire)
	}
}

// A server that sends no header leaves the traffic unknown.
func TestFetchWithoutUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`proxies:
  - {name: a, type: socks5, server: 127.0.0.1, port: 1080}
`))
	}))
	defer server.Close()
	_, info, err := Fetch(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Total != 0 || info.Upload != 0 || info.Download != 0 {
		t.Errorf("没有头却读到流量 %+v", info)
	}
}
