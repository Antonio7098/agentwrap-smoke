package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/ultraplan/agentwrap-smoke/internal/types"
)

var (
	UltraPlanRoot  = findUltraPlanRoot()
	StudiesDir     = filepath.Join(UltraPlanRoot, "studies")
	BackoffDelays  = []time.Duration{
		30 * time.Minute,
		1 * time.Hour,
		90 * time.Minute,
		2 * time.Hour,
		3 * time.Hour,
		5 * time.Hour,
		7 * time.Hour,
		9 * time.Hour,
		12 * time.Hour,
		15 * time.Hour,
		18 * time.Hour,
		24 * time.Hour,
	}
)

func findUltraPlanRoot() string {
	if envRoot := os.Getenv("ULTRAPLAN_ROOT"); envRoot != "" {
		return envRoot
	}
	cwd, _ := os.Getwd()
	return cwd
}

func Load(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		cfg := types.DefaultConfig
		return &cfg, nil
	}
	var cfg types.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		cfg := types.DefaultConfig
		return &cfg, nil
	}
	return &cfg, nil
}

func OpenCodeConfigPath() string {
	return filepath.Join(UltraPlanRoot, "opencode-config.json")
}