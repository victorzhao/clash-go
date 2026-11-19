package mihomoapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Delay sends its timeout in milliseconds, as the core's API reads it.
func TestDelayQuery(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"delay":123}`))
	}))
	defer server.Close()
	c := New(strings.TrimPrefix(server.URL, "http://"), "")
	delay, err := c.Delay("node", "https://www.binance.com", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if delay != 123 {
		t.Errorf("delay = %d, want 123", delay)
	}
	if !strings.Contains(query, "timeout=5000") {
		t.Errorf("query %q has no timeout=5000", query)
	}
	if !strings.Contains(query, "url=https%3A%2F%2Fwww.binance.com") {
		t.Errorf("query %q has no url", query)
	}
}
