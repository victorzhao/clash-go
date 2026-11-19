// Package profile downloads subscriptions and reads what their servers
// say about them.
package profile

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Info is what a subscription's server said about it.
type Info struct {
	// Interval is the profile's own update interval in hours, 0 for none.
	Interval int
	// Upload, Download and Total are the bytes used of the plan, and its
	// size, 0 when the server doesn't say.
	Upload, Download, Total int64
	// Expire is when the plan ends, zero when the server doesn't say.
	Expire time.Time
}

var client = &http.Client{Timeout: 45 * time.Second}

// Fetch downloads a subscription and checks that it is a Clash
// configuration.
func Fetch(url string) ([]byte, Info, error) {
	var info Info
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, info, err
	}
	req.Header.Set("User-Agent", "clash-go/1.0 (mihomo)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, info, fmt.Errorf("服务器返回 %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, info, err
	}
	if len(data) == 0 {
		return nil, info, errors.New("订阅内容为空")
	}
	if interval, err := strconv.Atoi(strings.TrimSpace(resp.Header.Get("profile-update-interval"))); err == nil {
		info.Interval = interval
	}
	info = parseUserInfo(resp.Header.Get("subscription-userinfo"), info)
	data, err = normalize(data)
	if err != nil {
		return nil, info, err
	}
	if err := valid(data); err != nil {
		return nil, info, err
	}
	return data, info, nil
}

// valid checks that data is a Clash configuration with proxies.
func valid(data []byte) error {
	var cfg struct {
		Proxies        []any `yaml:"proxies"`
		ProxyProviders any   `yaml:"proxy-providers"`
		ProxyGroups    []any `yaml:"proxy-groups"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("不是有效的 YAML：%w", err)
	}
	if len(cfg.Proxies) == 0 && cfg.ProxyProviders == nil {
		return errors.New("配置中没有代理")
	}
	return nil
}

func parseUserInfo(s string, info Info) Info {
	for _, part := range strings.Split(s, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(kv[0])) {
		case "upload":
			info.Upload = v
		case "download":
			info.Download = v
		case "total":
			info.Total = v
		case "expire":
			info.Expire = time.Unix(v, 0)
		}
	}
	return info
}

// Prepare normalizes share links into a Clash configuration and validates
// it, returning the YAML to save. Callers should store the returned bytes.
func Prepare(data []byte) ([]byte, error) {
	out, err := normalize(data)
	if err != nil {
		return nil, err
	}
	if err := valid(out); err != nil {
		return nil, err
	}
	return out, nil
}

// Check reports whether data is a Clash configuration with proxies, after
// share links have been converted.
func Check(data []byte) error {
	_, err := Prepare(data)
	return err
}
