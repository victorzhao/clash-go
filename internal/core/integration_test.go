package core

import (
	"testing"

	"github.com/victorzhao/clash-go/internal/mihomoapi"
	"github.com/victorzhao/clash-go/internal/store"
)

// Start runs mihomo in the process and its API answers, as the app needs.
func TestCoreStarts(t *testing.T) {
	m := New(t.TempDir())
	s := store.Default()
	s.Mode = "direct"
	s.MixedPort = 17890
	s.APIPort = 16170
	s.Secret = "test-secret"
	s.LogLevel = "warning"
	if err := m.Start([]byte("proxies: []\nrules:\n  - MATCH,DIRECT\n"), s); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	api := mihomoapi.New("127.0.0.1:16170", "test-secret")
	version, err := api.Version()
	if err != nil {
		t.Fatal(err)
	}
	if version == "" {
		t.Error("version 为空")
	}
	if running, _ := m.Running(); !running {
		t.Error("内核应处于运行状态")
	}
	if _, err := api.Connections(); err != nil {
		t.Errorf("读取连接失败: %v", err)
	}
	if _, err := api.Rules(); err != nil {
		t.Errorf("读取规则失败: %v", err)
	}
}
