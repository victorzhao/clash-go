package profile

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/metacubex/mihomo/common/convert"
	"gopkg.in/yaml.v3"
)

// shareSchemes are the proxy share-link schemes we accept. mihomo's
// converter understands all of them, including vless://.
var shareSchemes = []string{
	"vless://", "vmess://", "trojan://", "ss://", "ssr://",
	"hysteria://", "hysteria2://", "hy2://", "tuic://", "socks://",
	"socks5://", "anytls://", "http://", "https://",
}

// isShareLink reports whether buf looks like a list of proxy share links,
// either plain or base64-encoded (as subscriptions often are).
func isShareLink(buf []byte) bool {
	s := strings.TrimSpace(string(buf))
	if s == "" {
		return false
	}
	// Plain links: the first non-empty line starts with a known scheme.
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		for _, scheme := range shareSchemes {
			if strings.HasPrefix(lower, scheme) {
				return true
			}
		}
		// A non-YAML line that is not a share link is likely base64.
		return !looksLikeYAML(s)
	}
	return false
}

func looksLikeYAML(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// A YAML mapping or sequence starts with a key or a dash.
		return strings.Contains(line, ":") || strings.HasPrefix(line, "-")
	}
	return false
}

// ConvertShareLinks turns a list of proxy share links (plain or base64)
// into a minimal Clash configuration that mihomo can run: the proxies, a
// selector group holding them all, and a catch-all rule.
func ConvertShareLinks(buf []byte) ([]byte, error) {
	proxies, err := convert.ConvertsV2Ray(buf)
	if err != nil {
		return nil, fmt.Errorf("解析分享链接失败：%w", err)
	}
	if len(proxies) == 0 {
		return nil, errors.New("没有识别到代理链接（支持 vless:// 等格式）")
	}
	names := make([]string, 0, len(proxies))
	for _, p := range proxies {
		if name, ok := p["name"].(string); ok && name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, errors.New("分享链接没有名称")
	}
	cfg := map[string]any{
		"proxies": proxies,
		"proxy-groups": []any{
			map[string]any{
				"name":    "节点选择",
				"type":    "select",
				"proxies": names,
			},
		},
		"rules": []string{"MATCH,节点选择"},
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// normalize converts share-link input into a Clash YAML profile. Other
// input is returned unchanged. It is applied to subscriptions and pasted
// text so that both accept vless and friends.
func normalize(buf []byte) ([]byte, error) {
	if !isShareLink(buf) {
		return buf, nil
	}
	converted, err := ConvertShareLinks(buf)
	if err != nil {
		return nil, err
	}
	if !bytes.Contains(converted, []byte("proxies:")) {
		return nil, errors.New("转换后的配置没有代理")
	}
	return converted, nil
}
