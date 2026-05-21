package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/ultraplan/agentwrap-smoke/internal/types"
)

func Load(root string) (*types.RunState, error) {
	path := filepath.Join(root, ".run-state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var st types.RunState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, nil
	}
	return &st, nil
}

func Save(root string, st *types.RunState) error {
	st.UpdatedAt = time.Now()
	path := filepath.Join(root, ".run-state.json")
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}