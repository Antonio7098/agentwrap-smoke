package discover

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ultraplan/agentwrap-smoke/internal/types"
)

func Sources(root string) []types.Source {
	srcDir := filepath.Join(root, "sources")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil
	}
	var sources []types.Source
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		sources = append(sources, types.Source{
			Name: e.Name(),
			Path: filepath.Join(srcDir, e.Name()),
		})
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})
	return sources
}

func Dimensions(root string) []types.Dimension {
	dimDir := filepath.Join(root, "dimensions")
	entries, err := os.ReadDir(dimDir)
	if err != nil {
		return nil
	}
	var dims []types.Dimension
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		file := e.Name()
		dash := strings.Index(file, "-")
		var number, name string
		if dash > 0 {
			number = file[:dash]
			name = strings.TrimSuffix(file[dash+1:], ".md")
		} else {
			name = strings.TrimSuffix(file, ".md")
		}
		title := name
		content, _ := os.ReadFile(filepath.Join(dimDir, file))
		if lines := strings.Split(string(content), "\n"); len(lines) > 0 {
			title = strings.TrimPrefix(lines[0], "# ")
			title = strings.TrimSpace(title)
			if title == "" {
				title = name
			}
		}
		dims = append(dims, types.Dimension{
			Number: number,
			Name:   name,
			Title:  title,
			File:   file,
		})
	}
	sort.Slice(dims, func(i, j int) bool {
		return dims[i].File < dims[j].File
	})
	return dims
}

func ResolveDimension(ref string, all []types.Dimension) *types.Dimension {
	for _, d := range all {
		if d.Number == ref {
			return &d
		}
		key := d.Number + "-" + d.Name
		if key == ref {
			return &d
		}
		if strings.HasPrefix(key, ref) {
			return &d
		}
		if strings.HasPrefix(d.Name, ref) {
			return &d
		}
	}
	return nil
}

func ResolveSource(ref string, all []types.Source) *types.Source {
	for _, s := range all {
		if s.Name == ref {
			return &s
		}
		if strings.HasPrefix(s.Name, ref) {
			return &s
		}
	}
	return nil
}