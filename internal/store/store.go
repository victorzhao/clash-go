// Package store keeps the app's settings and profiles on disk: the
// configuration directory holds settings.json, profiles.json and a
// profiles/ directory with one YAML file for each profile.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Settings are the app's own settings, applied to the core when it starts.
type Settings struct {
	// Mode is the core's mode: rule, global or direct.
	Mode string `json:"mode"`
	// MixedPort is the port of the mixed HTTP/SOCKS listener.
	MixedPort int `json:"mixedPort"`
	// APIPort is the port of the core's RESTful API on the loopback
	// interface.
	APIPort int `json:"apiPort"`
	// Secret authenticates requests to the API.
	Secret string `json:"secret"`
	// AllowLAN lets other hosts on the network use the proxy.
	AllowLAN bool `json:"allowLan"`
	// IPv6 lets the core resolve and route IPv6.
	IPv6 bool `json:"ipv6"`
	// LogLevel is the core's log level: debug, info, warning or error.
	LogLevel string `json:"logLevel"`
	// TunEnabled routes all traffic through a virtual network device.
	TunEnabled bool   `json:"tunEnabled"`
	TunStack   string `json:"tunStack"`
	// SystemProxy sets the system's proxy to the mixed port while the core
	// runs.
	SystemProxy bool `json:"systemProxy"`
	// AutoStart starts the app when the user signs in.
	AutoStart bool `json:"autoStart"`
	// Theme is system, light or dark.
	Theme string `json:"theme"`
	// Language is the interface's language: zh-CN, zh-TW or en.
	Language string `json:"language"`
	// ActiveProfile is the ID of the profile the core runs.
	ActiveProfile string `json:"activeProfile"`
	// UpdateInterval is how many hours between subscription updates, 0 for
	// none.
	UpdateInterval int `json:"updateInterval"`
	// StartCore starts the core as the app opens.
	StartCore bool `json:"startCore"`
	// TestURL is the website the dashboard checks for connectivity through
	// the selected proxy.
	TestURL string `json:"testUrl"`
}

// Profile is a subscription or a local configuration file.
type Profile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // remote or local
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updatedAt"`
	// Interval is the profile's own update interval in hours, from its
	// subscription headers, 0 for none.
	Interval int `json:"interval"`
	// Used and Total are the bytes the subscription says are used of its
	// plan, and its size; Expire is when the plan ends. Zero when the
	// server doesn't say.
	Used   int64     `json:"used,omitempty"`
	Total  int64     `json:"total,omitempty"`
	Expire time.Time `json:"expire,omitempty"`
}

// Default returns the settings of a first run.
func Default() Settings {
	return Settings{
		Mode:           "rule",
		MixedPort:      7890,
		APIPort:        6170,
		Secret:         newSecret(),
		LogLevel:       "info",
		TunStack:       "mixed",
		Theme:          "system",
		Language:       "zh-CN",
		UpdateInterval: 0,
		StartCore:      true,
		TestURL:        "http://www.gstatic.com/generate_204",
	}
}

func newSecret() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "clash-go"
	}
	return hex.EncodeToString(b[:])
}

// Store reads and writes the app's files.
type Store struct {
	dir string
	mu  sync.Mutex
}

// Open returns the store of the user's configuration directory, creating
// it if needed.
func Open() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return OpenAt(filepath.Join(dir, "clash-go"))
}

// OpenAt returns the store of dir, creating it if needed.
func OpenAt(dir string) (*Store, error) {
	s := &Store{dir: dir}
	for _, sub := range []string{s.dir, s.profileDir()} {
		if err := os.MkdirAll(sub, 0o755); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Dir returns the store's directory, the core's home.
func (s *Store) Dir() string { return s.dir }

func (s *Store) profileDir() string { return filepath.Join(s.dir, "profiles") }

func (s *Store) settingsPath() string { return filepath.Join(s.dir, "settings.json") }
func (s *Store) profilesPath() string { return filepath.Join(s.dir, "profiles.json") }

// Load returns the settings and profiles on disk, with the defaults where
// none are saved yet.
func (s *Store) Load() (Settings, []Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := Default()
	data, err := os.ReadFile(s.settingsPath())
	if err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return settings, nil, fmt.Errorf("settings.json: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return settings, nil, err
	}
	if settings.Secret == "" {
		settings.Secret = newSecret()
	}
	if settings.MixedPort == 0 {
		settings.MixedPort = 7890
	}
	if settings.APIPort == 0 {
		settings.APIPort = 6170
	}
	var profiles []Profile
	data, err = os.ReadFile(s.profilesPath())
	if err == nil {
		if err := json.Unmarshal(data, &profiles); err != nil {
			return settings, nil, fmt.Errorf("profiles.json: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return settings, nil, err
	}
	sort.SliceStable(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	return settings, profiles, nil
}

// SaveSettings writes the settings.
func (s *Store) SaveSettings(v Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(s.settingsPath(), data)
}

// SaveProfiles writes the profiles' metadata.
func (s *Store) SaveProfiles(v []Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(s.profilesPath(), data)
}

func (s *Store) profilePath(id string) string {
	return filepath.Join(s.profileDir(), id+".yaml")
}

// WriteProfile saves a profile's configuration.
func (s *Store) WriteProfile(id string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeFile(s.profilePath(id), data)
}

// ProfileContent reads a profile's configuration.
func (s *Store) ProfileContent(id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.ReadFile(s.profilePath(id))
}

// DeleteProfile removes a profile's configuration.
func (s *Store) DeleteProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.profilePath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// NewProfileID returns an identifier for a new profile.
func NewProfileID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprint(time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func writeFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
