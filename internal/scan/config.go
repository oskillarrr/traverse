package scan

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the optional per-portfolio or per-application configuration file
// (traverse.json). Everything is optional.
type Config struct {
	Exclude []string `json:"exclude"` // path globs relative to repo root
	Layers  struct {
		Overrides map[string]string   `json:"overrides"` // package/FQN prefix -> layer
		Tokens    map[string][]string `json:"tokens"`    // extra package tokens per layer
	} `json:"layers"`
	Classification string `json:"classification"` // banner printed on every report
	// Rules lists extra rule files or packs (for example a regional pack from
	// rulepacks/), relative to this configuration file. They load after the
	// core catalog and before any --rules given on the command line.
	Rules []string `json:"rules"`
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := json.Unmarshal(b, c); err != nil {
		return nil, err
	}
	for i, r := range c.Rules {
		if !filepath.IsAbs(r) {
			c.Rules[i] = filepath.Join(filepath.Dir(path), r)
		}
	}
	return c, nil
}
