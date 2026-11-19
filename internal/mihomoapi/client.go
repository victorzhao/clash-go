// Package mihomoapi talks to the core's RESTful API on the loopback
// interface: proxies and their groups, connections, rules, traffic and
// delay tests.
package mihomoapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client calls the core's API.
type Client struct {
	base   string
	secret string
	http   *http.Client
}

// New returns a client for the API at addr ("127.0.0.1:6170") with secret.
func New(addr, secret string) *Client {
	return &Client{
		base:   "http://" + addr,
		secret: secret,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(method, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, r)
	if err != nil {
		return err
	}
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &e) == nil && e.Message != "" {
			return fmt.Errorf("%s", e.Message)
		}
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	return nil
}

// History is a proxy's delay test result.
type History struct {
	Time  time.Time `json:"time"`
	Delay uint16    `json:"delay"`
}

// Proxy is a proxy or a group of them.
type Proxy struct {
	Name    string    `json:"name"`
	Type    string    `json:"type"`
	Now     string    `json:"now"`
	All     []string  `json:"all"`
	History []History `json:"history"`
	UDP     bool      `json:"udp"`
	Alive   bool      `json:"alive"`
	// Delay is the last delay in milliseconds, from History.
	Delay int `json:"-"`
}

// Proxies returns the core's proxies and groups by name.
func (c *Client) Proxies() (map[string]*Proxy, error) {
	var resp struct {
		Proxies map[string]*Proxy `json:"proxies"`
	}
	if err := c.do(http.MethodGet, "/proxies", nil, &resp); err != nil {
		return nil, err
	}
	for _, p := range resp.Proxies {
		if n := len(p.History); n > 0 {
			p.Delay = int(p.History[n-1].Delay)
		}
	}
	return resp.Proxies, nil
}

// Select makes group use the proxy named name.
func (c *Client) Select(group, name string) error {
	return c.do(http.MethodPut, "/proxies/"+url.PathEscape(group), map[string]string{"name": name}, nil)
}

// Delay tests a proxy and returns its delay in milliseconds.
func (c *Client) Delay(name, testURL string, timeout time.Duration) (int, error) {
	q := url.Values{}
	q.Set("url", testURL)
	// time.Duration's String would write "5µs" here: send milliseconds.
	q.Set("timeout", strconv.FormatInt(int64(timeout/time.Millisecond), 10))
	var resp struct {
		Delay int `json:"delay"`
	}
	if err := c.do(http.MethodGet, "/proxies/"+url.PathEscape(name)+"/delay?"+q.Encode(), nil, &resp); err != nil {
		return 0, err
	}
	return resp.Delay, nil
}

// GroupDelay tests every proxy of a group, returning their delays by name.
func (c *Client) GroupDelay(name, testURL string, timeout time.Duration) (map[string]int, error) {
	q := url.Values{}
	q.Set("url", testURL)
	// time.Duration's String would write "5µs" here: send milliseconds.
	q.Set("timeout", strconv.FormatInt(int64(timeout/time.Millisecond), 10))
	var resp map[string]int
	if err := c.do(http.MethodGet, "/group/"+url.PathEscape(name)+"/delay?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Traffic is the core's traffic since the last request, in bytes.
type Traffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// Traffic returns the traffic since the last call.
func (c *Client) Traffic() (*Traffic, error) {
	var t Traffic
	if err := c.do(http.MethodGet, "/traffic", nil, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// Metadata is what a connection connects.
type Metadata struct {
	Network     string `json:"network"`
	Type        string `json:"type"`
	Host        string `json:"host"`
	SourceIP    string `json:"sourceIP"`
	Destination string `json:"destinationIP"`
	SourcePort  string `json:"sourcePort"`
	DstPort     string `json:"destinationPort"`
	Process     string `json:"process"`
	ProcessPath string `json:"processPath"`
	Rule        string `json:"-"`
}

// Connection is one connection the core carries.
type Connection struct {
	ID       string    `json:"id"`
	Metadata Metadata  `json:"metadata"`
	Upload   int64     `json:"upload"`
	Download int64     `json:"download"`
	Start    time.Time `json:"start"`
	Chains   []string  `json:"chains"`
	Rule     string    `json:"rule"`
	Payload  string    `json:"rulePayload"`
}

// Connections is the core's connections.
type Connections struct {
	DownloadTotal int64        `json:"downloadTotal"`
	UploadTotal   int64        `json:"uploadTotal"`
	Memory        uint64       `json:"memory"`
	Connections   []Connection `json:"connections"`
}

// Connections returns the core's connections.
func (c *Client) Connections() (*Connections, error) {
	var v Connections
	if err := c.do(http.MethodGet, "/connections", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// CloseAllConnections closes every connection.
func (c *Client) CloseAllConnections() error {
	return c.do(http.MethodDelete, "/connections", nil, nil)
}

// CloseConnection closes the connection with id.
func (c *Client) CloseConnection(id string) error {
	return c.do(http.MethodDelete, "/connections/"+url.PathEscape(id), nil, nil)
}

// Rule is a rule of the profile.
type Rule struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
	Size    int    `json:"size"`
}

// Rules returns the core's rules.
func (c *Client) Rules() ([]Rule, error) {
	var resp struct {
		Rules []Rule `json:"rules"`
	}
	if err := c.do(http.MethodGet, "/rules", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Rules, nil
}

// Version returns the core's version.
func (c *Client) Version() (string, error) {
	var resp struct {
		Version string `json:"version"`
		Meta    bool   `json:"meta"`
	}
	if err := c.do(http.MethodGet, "/version", nil, &resp); err != nil {
		return "", err
	}
	return resp.Version, nil
}
