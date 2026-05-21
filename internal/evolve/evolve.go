package evolve

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ultraplan/agentwrap-smoke/internal/code"
	"github.com/ultraplan/agentwrap-smoke/internal/types"
)

var (
	sourceReportRE = regexp.MustCompile(`^\s*-\s*` + "`([^`]+)`" + `\s*$`)
	scoreRE        = regexp.MustCompile(`\*\*(\d+(?:\.\d+)?)\s*\/\s*10\s*\*\*`)
)

type EvolveOptions struct {
	TopSources int
	OutputFile string
	NoCode     bool
	FinalOnly  bool
}

type ReportEntry struct {
	Type  string
	Label string
	Path  string
	Lines int
	Chars int
}

type RefStats struct {
	Total            int
	Rendered         int
	DuplicateSkipped int
	Resolved         int
	Unresolved       int
	UnresolvedMdRefs int
	UnresolvedCode   int
}

func CmdEvolve(fileArgs []string, opts EvolveOptions) {
	if len(fileArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: study evolve [--top-sources <N>] [--output <file>] [--no-code] [--final-only] <@evidence-report>...")
		os.Exit(1)
	}

	resolvedPaths := make([]string, len(fileArgs))
	for i, a := range fileArgs {
		stripped := strings.TrimPrefix(a, "@")
		if filepath.IsAbs(stripped) {
			resolvedPaths[i] = stripped
		} else {
			cwd, _ := os.Getwd()
			resolvedPaths[i], _ = filepath.Abs(filepath.Join(cwd, stripped))
		}
	}

	for _, p := range resolvedPaths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: evidence report not found: %s\n", p)
			os.Exit(1)
		}
	}

	var outputParts []string
	stats := RefStats{}
	reportLog := []ReportEntry{}
	seenFinalPaths := make(map[string]bool)
	seenPerSourcePaths := make(map[string]bool)
	seenCodeKeys := make(map[string]bool)

	seenEvidence := make(map[string]bool)
	var uniqueEvidence []struct {
		path    string
		content string
	}
	var allFinalReports []struct {
		path  string
		label string
		typ   string
	}

	for _, evPath := range resolvedPaths {
		if seenEvidence[evPath] {
			continue
		}
		seenEvidence[evPath] = true
		content := readReport(evPath)
		uniqueEvidence = append(uniqueEvidence, struct {
			path    string
			content string
		}{evPath, content})
		srcReports := parseSourceReports(content)
		for _, sr := range srcReports {
			key := sr.path
			if !seenFinalPaths[key] {
				seenFinalPaths[key] = true
				allFinalReports = append(allFinalReports, sr)
			}
		}
	}

	type finalWithPerSource struct {
		sr       struct {
			path  string
			label string
			typ   string
		}
		perSource []struct {
			path  string
			name  string
			score float64
		}
	}
	var finalReportsWithPerSource []finalWithPerSource
	for _, sr := range allFinalReports {
		pss := findPerSourceReports(sr.path, opts.TopSources)
		filtered := make([]struct {
			path  string
			name  string
			score float64
		}, 0, len(pss))
		for _, ps := range pss {
			key := ps.path
			if seenPerSourcePaths[key] {
				continue
			}
			seenPerSourcePaths[key] = true
			filtered = append(filtered, ps)
		}
		finalReportsWithPerSource = append(finalReportsWithPerSource, finalWithPerSource{
			sr:       sr,
			perSource: filtered,
		})
	}

	outputParts = append(outputParts,
		"════════════════════════════════════════════════════════",
		"Planning Load Order",
		"════════════════════════════════════════════════════════",
		"",
		"This bundle is a staged planning source, not a request to load every report at once.",
		"",
		"1. Read the evidence packs and the selected roadmap sprint section first.",
		"2. Read final reports below only for decisions in the sprint scope.",
		"3. Use the per-source manifest to open individual source reports only when a final report is not specific enough.",
		"4. Resolve code references or inspect repository code only for concrete implementation questions.",
		"",
		"Included final reports:")
	for _, sr := range allFinalReports {
		outputParts = append(outputParts, fmt.Sprintf("- [%s] %s", sr.typ, sr.label))
	}
	outputParts = append(outputParts, "", "Available per-source reports:")
	for _, fwps := range finalReportsWithPerSource {
		if len(fwps.perSource) == 0 {
			continue
		}
		outputParts = append(outputParts, fmt.Sprintf("- %s", fwps.sr.label))
		for _, ps := range fwps.perSource {
			outputParts = append(outputParts, fmt.Sprintf("  - %s (%.0f/10): %s", ps.name, ps.score, ps.path))
		}
	}
	outputParts = append(outputParts, "")

	for _, ev := range uniqueEvidence {
		evContent := ev.content
		reportLog = append(reportLog, ReportEntry{
			Type:  "evidence",
			Label: strings.TrimSuffix(filepath.Base(ev.path), ".md"),
			Path:  ev.path,
			Lines: strings.Count(evContent, "\n") + 1,
			Chars: len(evContent),
		})
		outputParts = append(outputParts,
			"════════════════════════════════════════════════════════",
			"Evidence Pack: "+strings.TrimSuffix(filepath.Base(ev.path), ".md"),
			"File: "+ev.path,
			"════════════════════════════════════════════════════════",
			"",
			strings.TrimRight(evContent, "\n"),
			"")
	}

	for _, fwps := range finalReportsWithPerSource {
		sr := fwps.sr
		label := "[PRIMARY] " + sr.label
		if sr.typ == "supporting" {
			label = "[SUPPORTING] " + sr.label
		}
		srContent := readReport(sr.path)
		reportLog = append(reportLog, ReportEntry{
			Type:  "final",
			Label: sr.label,
			Path:  sr.path,
			Lines: strings.Count(srContent, "\n") + 1,
			Chars: len(srContent),
		})

		outputParts = append(outputParts,
			"════════════════════════════════════════════════════════",
			"Final Report: "+label,
			"File: "+sr.path,
			"════════════════════════════════════════════════════════",
			"",
			strings.TrimRight(srContent, "\n"),
			"")

		if !opts.NoCode && !opts.FinalOnly {
			finalRepos := code.ParseReposTable(srContent, filepath.Dir(sr.path))
			refs := code.FindCodeRefs(srContent, finalRepos, sr.path)
			codeOutput := dedupedCode(refs, &stats, seenCodeKeys)
			if strings.TrimSpace(codeOutput) != "" {
				outputParts = append(outputParts,
					"────────────────────────────────────────────────────────",
					"Code References from "+sr.label,
					"────────────────────────────────────────────────────────",
					"",
					strings.TrimRight(codeOutput, "\n"),
					"")
			}
		}

		if len(fwps.perSource) > 0 && !opts.FinalOnly {
			limitLabel := ""
			if opts.TopSources > 0 && opts.TopSources < len(fwps.perSource) {
				limitLabel = fmt.Sprintf(" (top %d by score)", opts.TopSources)
			}
			outputParts = append(outputParts,
				"────────────────────────────────────────────────────────",
				"Per-Source Reports"+limitLabel+":",
				"────────────────────────────────────────────────────────",
				"")

			for _, ps := range fwps.perSource {
				outputParts = append(outputParts,
					fmt.Sprintf("--- %s (%.0f/10) ---", ps.name, ps.score),
					"File: "+ps.path,
					"")
				psContent := readReport(ps.path)
				reportLog = append(reportLog, ReportEntry{
					Type:  "per-source",
					Label: fmt.Sprintf("%s (%.0f/10)", ps.name, ps.score),
					Path:  ps.path,
					Lines: strings.Count(psContent, "\n") + 1,
					Chars: len(psContent),
				})
				outputParts = append(outputParts, strings.TrimRight(psContent, "\n"), "")

				if !opts.NoCode {
					finalRepos := code.ParseReposTable(srContent, filepath.Dir(sr.path))
					var sourceRepoEntry *types.RepoEntry
					for _, r := range finalRepos {
						if r.Name == ps.name {
							sourceRepoEntry = &r
							break
						}
					}
					contextRepos := finalRepos
					if sourceRepoEntry != nil {
						contextRepos = []types.RepoEntry{*sourceRepoEntry}
					}
					refs := code.FindCodeRefs(psContent, contextRepos, ps.path)
					codeOutput := dedupedCode(refs, &stats, seenCodeKeys)
					if strings.TrimSpace(codeOutput) != "" {
						outputParts = append(outputParts, strings.TrimRight(codeOutput, "\n"), "")
					}
				}
			}
		}
	}

	finalOutputStr := strings.Join(outputParts, "\n")
	totalLines := strings.Count(finalOutputStr, "\n") + 1
	totalChars := len(finalOutputStr)
	estimatedTokens := (totalChars + 3) / 4

	evidenceCount := 0
	finalCount := 0
	perSourceCount := 0
	for _, r := range reportLog {
		switch r.Type {
		case "evidence":
			evidenceCount++
		case "final":
			finalCount++
		case "per-source":
			perSourceCount++
		}
	}

	outputParts = append(outputParts,
		"════════════════════════════════════════════════════════",
		"Bundle Summary",
		"════════════════════════════════════════════════════════",
		"",
		"Reports included:",
		fmt.Sprintf("  %d evidence pack(s)", evidenceCount),
		fmt.Sprintf("  %d final report(s)", finalCount),
		fmt.Sprintf("  %d per-source report(s)", perSourceCount))
	if opts.FinalOnly {
		availablePerSource := 0
		for _, fwps := range finalReportsWithPerSource {
			availablePerSource += len(fwps.perSource)
		}
		outputParts = append(outputParts, fmt.Sprintf("  %d per-source report(s) listed in manifest, not injected", availablePerSource))
	}
	outputParts = append(outputParts,
		"",
		fmt.Sprintf("  Total lines:        %d", totalLines),
		fmt.Sprintf("  Total characters:   %d", totalChars),
		fmt.Sprintf("  Estimated tokens:   %d  (~4 chars/token)", estimatedTokens),
		"")
	if opts.FinalOnly {
		outputParts = append(outputParts, "Code reference resolution: skipped (--final-only)")
	} else {
		outputParts = append(outputParts,
			"Code reference resolution:",
			fmt.Sprintf("  Total refs found:   %d", stats.Total),
			fmt.Sprintf("  Rendered unique:    %d", stats.Rendered),
			fmt.Sprintf("  Duplicates skipped: %d", stats.DuplicateSkipped),
			fmt.Sprintf("  Resolved:           %d", stats.Resolved),
			fmt.Sprintf("  Unresolved (total): %d", stats.Unresolved),
			fmt.Sprintf("    ├─ .md self-refs:  %d  (cross-refs to analysis files, not code)", stats.UnresolvedMdRefs),
			fmt.Sprintf("    └─ code refs:      %d", stats.UnresolvedCode))
		pct := "0.0"
		if stats.Total > 0 {
			pct = fmt.Sprintf("%.1f", float64(stats.Resolved)/float64(stats.Total)*100)
		}
		outputParts = append(outputParts, fmt.Sprintf("  Resolution rate:    %s%%", pct))
	}
	outputParts = append(outputParts, "")

	finalOutput := strings.Join(outputParts, "\n")

	if opts.OutputFile != "" {
		os.WriteFile(opts.OutputFile, []byte(finalOutput), 0644)
		fmt.Fprintf(os.Stderr, "Evolved output written to %s\n", opts.OutputFile)
	} else {
		fmt.Println(finalOutput)
	}
}

func dedupedCode(refs []types.CodeRef, stats *RefStats, seenKeys map[string]bool) string {
	var parts []string
	for _, ref := range refs {
		stats.Total++
		if ref.RepoName == "???" {
			stats.Unresolved++
			if strings.HasSuffix(ref.FilePath, ".md") {
				stats.UnresolvedMdRefs++
			} else {
				stats.UnresolvedCode++
			}
		} else {
			stats.Resolved++
		}
		key := ref.FullPath + ":" + ref.LineSpec
		if key == ":" {
			key = "unresolved:" + ref.FilePath + ":" + ref.LineSpec
		}
		if seenKeys[key] {
			stats.DuplicateSkipped++
			continue
		}
		seenKeys[key] = true
		stats.Rendered++
		parts = append(parts, code.ReadCode(ref))
	}
	return strings.Join(parts, "")
}

func parseSourceReports(content string) []struct {
	path  string
	label string
	typ   string
} {
	var reports []struct {
		path  string
		label string
		typ   string
	}
	lines := strings.Split(content, "\n")
	inSourceReports := false
	var currentTyp string

	for _, line := range lines {
		if strings.TrimSpace(line) == "## Source Reports" {
			inSourceReports = true
			continue
		}
		if !inSourceReports {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			break
		}
		if strings.TrimSpace(line) == "Primary:" {
			currentTyp = "primary"
			continue
		}
		if strings.TrimSpace(line) == "Supporting:" {
			currentTyp = "supporting"
			continue
		}
		if currentTyp != "" {
			m := sourceReportRE.FindStringSubmatch(line)
			if m != nil {
				reports = append(reports, struct {
					path  string
					label string
					typ   string
				}{m[1], m[1], currentTyp})
			}
		}
	}
	return reports
}

func readReport(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "// REPORT NOT FOUND: " + path + "\n\n"
	}
	return string(data)
}

func findPerSourceReports(finalReportPath string, topN int) []struct {
	path  string
	name  string
	score float64
} {
	relPath := finalReportPath
	if strings.Contains(relPath, "/studies/") {
		idx := strings.Index(relPath, "/studies/")
		if idx >= 0 {
			relPath = relPath[idx+len("/studies/"):]
		}
	}
	parts := strings.Split(relPath, "/")
	if len(parts) < 2 {
		return nil
	}
	dimPart := parts[len(parts)-1]
	dimPart = strings.TrimSuffix(dimPart, ".md")

	studyIdx := -1
	for i, p := range parts {
		if p == "studies" && i+1 < len(parts) {
			studyIdx = i + 1
			break
		}
	}
	if studyIdx == -1 {
		return nil
	}

	sourceDir := filepath.Join(parts[:studyIdx+2]...)
	sourceDir = filepath.Join(sourceDir, "reports", "source", dimPart)

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil
	}

	entries, _ := os.ReadDir(sourceDir)
	var reports []struct {
		path  string
		name  string
		score float64
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fullPath := filepath.Join(sourceDir, e.Name())
		content, _ := os.ReadFile(fullPath)
		score := extractScore(string(content))
		reports = append(reports, struct {
			path  string
			name  string
			score float64
		}{fullPath, strings.TrimSuffix(e.Name(), ".md"), score})
	}

	for i := range reports {
		for j := i + 1; j < len(reports); j++ {
			if reports[j].score > reports[i].score {
				reports[i], reports[j] = reports[j], reports[i]
			}
		}
	}

	if topN > 0 && topN < len(reports) {
		reports = reports[:topN]
	}
	return reports
}

func extractScore(content string) float64 {
	m := scoreRE.FindStringSubmatch(content)
	if m == nil {
		return 0
	}
	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	return f
}