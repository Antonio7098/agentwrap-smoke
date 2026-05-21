package code

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ultraplan/agentwrap-smoke/internal/types"
)

var (
	codeRefRE      = regexp.MustCompile("`([a-zA-Z_/][\\w./-]*\\.[a-zA-Z]\\w*):(\\d+(?:[-–,]\\d+)*)`")
	repoTableLineRE = regexp.MustCompile(`^\s*\|\s*\d+\s*\|\s*([^|]+?)\s*\|\s*` + "`([^`]+)`" + `\s*\|`)
)

var searchCache = make(map[string]string)

func ParseReposTable(content string, basePath string) []types.RepoEntry {
	var repos []types.RepoEntry
	for _, line := range strings.Split(content, "\n") {
		m := repoTableLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		repoPath := strings.TrimSpace(m[2])
		if !filepath.IsAbs(repoPath) && basePath != "" {
			resolved := filepath.Join(basePath, repoPath)
			if _, err := os.Stat(resolved); err == nil {
				repoPath = resolved
			}
		}
		repos = append(repos, types.RepoEntry{
			Name: strings.TrimSpace(m[1]),
			Path: repoPath,
		})
	}
	return repos
}

func FindCodeRefs(content string, repos []types.RepoEntry, reportPath string) []types.CodeRef {
	var refs []types.CodeRef
	seen := make(map[string]bool)

	matches := codeRefRE.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		filePath := match[1]
		lineSpec := match[2]
		key := filePath + ":" + lineSpec
		if seen[key] {
			continue
		}
		seen[key] = true

		resolved := false
		for _, repo := range repos {
			candidates := []string{filepath.Join(repo.Path, filePath)}
			if idx := strings.Index(filePath, "/"); idx > 0 {
				candidates = append(candidates, filepath.Join(repo.Path, filePath[idx+1:]))
			}
			for _, fullPath := range candidates {
				if _, err := os.Stat(fullPath); err == nil {
					refs = append(refs, types.CodeRef{
						RepoName:     repo.Name,
						FullPath:     fullPath,
						FilePath:     filePath,
						LineSpec:     lineSpec,
						SourceReport: reportPath,
					})
					resolved = true
					break
				}
			}
			if resolved {
				break
			}
		}

		if !resolved {
			fileName := filepath.Base(filePath)
			for _, repo := range repos {
				if found := searchFileInRepo(repo.Path, fileName); found != "" {
					rel, _ := filepath.Rel(repo.Path, found)
					refs = append(refs, types.CodeRef{
						RepoName:     repo.Name,
						FullPath:     found,
						FilePath:     rel,
						LineSpec:     lineSpec,
						SourceReport: reportPath,
					})
					resolved = true
					break
				}
			}
		}

		if !resolved {
			refs = append(refs, types.CodeRef{
				RepoName:     "???",
				FullPath:     "",
				FilePath:     filePath,
				LineSpec:     lineSpec,
				SourceReport: reportPath,
			})
		}
	}
	return refs
}

func searchFileInRepo(repoPath, fileName string) string {
	cacheKey := repoPath + "::" + fileName
	if cached, ok := searchCache[cacheKey]; ok {
		return cached
	}

	var search func(dir string) string
	search = func(dir string) string {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return ""
		}
		for _, e := range entries {
			if e.IsDir() {
				name := e.Name()
				if strings.HasPrefix(name, ".") || name == "node_modules" || name == ".git" {
					continue
				}
				if result := search(filepath.Join(dir, name)); result != "" {
					return result
				}
			} else if e.Name() == fileName {
				return filepath.Join(dir, e.Name())
			}
		}
		return ""
	}

	result := search(repoPath)
	searchCache[cacheKey] = result
	return result
}

func ReadCode(ref types.CodeRef) string {
	if ref.FullPath == "" {
		return "// ??? " + ref.FilePath + ":" + ref.LineSpec + " (unresolved)\n\n"
	}

	content, err := os.ReadFile(ref.FullPath)
	if err != nil {
		return "// ??? " + ref.FilePath + ":" + ref.LineSpec + " (read error)\n\n"
	}
	lines := strings.Split(string(content), "\n")

	startLine := 1
	endLine := len(lines)

	spec := ref.LineSpec
	if strings.Contains(spec, "–") {
		parts := strings.Split(spec, "–")
		if n := parseInt(parts[0]); n > 0 {
			startLine = n
		}
		if n := parseInt(parts[1]); n > 0 {
			endLine = n
		}
	} else if strings.Contains(spec, "-") {
		parts := strings.Split(spec, "-")
		if n := parseInt(parts[0]); n > 0 {
			startLine = n
		}
		if n := parseInt(parts[1]); n > 0 {
			endLine = n
		}
	} else if strings.Contains(spec, ",") {
		nums := splitToInts(spec, ",")
		if len(nums) > 0 {
			startLine = minOf(nums...)
			endLine = maxOf(nums...)
		}
	} else {
		if n := parseInt(spec); n > 0 {
			startLine = n
			endLine = min(len(lines), n+20)
		}
	}

	startLine = max(1, startLine)
	endLine = min(len(lines), endLine)

	selected := lines[startLine-1 : endLine]
	header := "// " + ref.RepoName + " / " + ref.FilePath + ":" + ref.LineSpec + "\n"
	var sb strings.Builder
	sb.WriteString(header)
	for i, line := range selected {
		sb.WriteString(fmt.Sprintf("%d  %s\n", startLine+i, line))
	}
	sb.WriteString("\n")
	return sb.String()
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return n
}

func splitToInts(s, sep string) []int {
	var nums []int
	for _, p := range strings.Split(s, sep) {
		if n := parseInt(strings.TrimSpace(p)); n > 0 {
			nums = append(nums, n)
		}
	}
	return nums
}

func minOf(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxOf(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func ProcessReports(reportPaths []string) string {
	var allRefs []types.CodeRef
	for _, rp := range reportPaths {
		content, err := os.ReadFile(rp)
		if err != nil {
			continue
		}
		reportDir := filepath.Dir(rp)
		repos := ParseReposTable(string(content), reportDir)
		refs := FindCodeRefs(string(content), repos, rp)
		allRefs = append(allRefs, refs...)
	}

	var sb strings.Builder
	for _, ref := range allRefs {
		sb.WriteString(ReadCode(ref))
	}
	return sb.String()
}