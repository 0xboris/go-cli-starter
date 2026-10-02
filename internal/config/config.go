// Package config resolves the config directory, loads a commented YAML config and
// writes it atomically. Precedence for every setting: flag > env > config > default.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Env var names; document every one in `tool help environment`.
const (
	EnvConfigDir      = "TOOL_CONFIG_DIR"
	EnvPager          = "TOOL_PAGER"
	EnvPromptDisabled = "TOOL_PROMPT_DISABLED"
	EnvDebug          = "TOOL_DEBUG"
	EnvBrowser        = "TOOL_BROWSER"
)

// Option declares a config key. The table drives validation, `config list`
// and the `config --help` text, so docs cannot drift from behavior.
type Option struct {
	Key           string
	Description   string
	DefaultValue  string
	AllowedValues []string
}

var Options = []Option{
	{Key: "prompt", Description: "toggle interactive prompting in the terminal", DefaultValue: "enabled", AllowedValues: []string{"enabled", "disabled"}},
	{Key: "pager", Description: "the terminal pager program to send standard output to", DefaultValue: ""},
}

type Config struct {
	path   string
	values map[string]string
}

// Dir returns TOOL_CONFIG_DIR > $XDG_CONFIG_HOME/tool > OS default.
func Dir() (string, error) {
	if d := os.Getenv(EnvConfigDir); d != "" {
		return d, nil
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "tool"), nil
	}
	if runtime.GOOS == "windows" {
		if d := os.Getenv("AppData"); d != "" {
			return filepath.Join(d, "Tool"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	return filepath.Join(home, ".config", "tool"), nil
}

// Load reads config.yml. A missing file is not an error.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	c := &Config{path: filepath.Join(dir, "config.yml"), values: map[string]string{}}
	data, err := os.ReadFile(c.path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &c.values); err != nil {
		return nil, fmt.Errorf("invalid config file %s: %w", c.path, err)
	}
	return c, nil
}

// NewFromMap builds an in-memory config for tests.
func NewFromMap(values map[string]string) *Config {
	return &Config{values: values}
}

// GetOrDefault returns the configured value or the option's default.
func (c *Config) GetOrDefault(key string) string {
	if v, ok := c.values[key]; ok {
		return v
	}
	for _, o := range Options {
		if o.Key == key {
			return o.DefaultValue
		}
	}
	return ""
}

// Set validates against AllowedValues.
func (c *Config) Set(key, value string) error {
	for _, o := range Options {
		if o.Key != key {
			continue
		}
		if len(o.AllowedValues) > 0 && !contains(o.AllowedValues, value) {
			return fmt.Errorf("invalid value %q for %s; valid values: %v", value, key, o.AllowedValues)
		}
		c.values[key] = value
		return nil
	}
	return fmt.Errorf("unknown config key %q", key)
}

// Save writes atomically: temp file in the same dir, then rename, preserving 0600.
func (c *Config) Save() error {
	if c.path == "" {
		return errors.New("in-memory config cannot be saved")
	}
	data, err := yaml.Marshal(c.values)
	if err != nil {
		return err
	}
	return WriteFileAtomic(c.path, data, 0o600)
}

// WriteFileAtomic replaces path with data without ever leaving a partial file.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target // replace the file, not the symlink
	}
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
