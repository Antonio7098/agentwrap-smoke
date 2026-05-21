package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ultraplan/agentwrap-smoke/internal/types"
)

func BuildAnalysisPrompt(root string, dim types.Dimension, source types.Source) string {
	dimFile := filepath.Join(root, "dimensions", dim.File)
	templateFile := filepath.Join(root, "templates", "repo-analysis.md")
	baseFile := filepath.Join(root, "prompts", "base.md")
	outputFile := fmt.Sprintf("reports/source/%s-%s/%s.md", dim.Number, dim.Name, source.Name)

	baseContent := readFile(baseFile)
	dimContent := readFile(dimFile)
	templateContent := readFile(templateFile)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Study: %s — %s\n\n", dim.Title, source.Name))
	sb.WriteString(fmt.Sprintf("Study **%s** following the instructions below.\n\n", source.Name))
	sb.WriteString("## Execution Instructions\n\n")
	sb.WriteString(baseContent)
	if baseContent == "" {
		sb.WriteString("(no base instructions)")
	}
	sb.WriteString("\n\n## Study Dimension\n\n")
	sb.WriteString(dimContent)
	if dimContent == "" {
		sb.WriteString("(no dimension content)")
	}
	sb.WriteString("\n\n## Target Source\n\n")
	sb.WriteString(fmt.Sprintf("1. **%s** (`%s`)\n\n", source.Name, source.Path))
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("1. Follow the Execution Instructions above.\n")
	sb.WriteString("2. Follow the Study Dimension above for the specific Steps, Evidence, and Questions.\n")
	sb.WriteString("3. **HARD RULES**:\n")
	sb.WriteString("   - When studying a source, NEVER access files outside that source's directory. BANNED.\n")
	sb.WriteString("   - EVERY code mention MUST include `path/to/file.ts:NN`. No exceptions.\n")
	sb.WriteString("4. Explore the source's code following the Study Dimension's Steps and Evidence sections.\n")
	sb.WriteString("   Answer all the Study Dimension's Questions.\n")
	sb.WriteString(fmt.Sprintf("5. Write the analysis to `%s` using the Output Template below.\n\n", outputFile))
	sb.WriteString("## Output Template\n\n")
	sb.WriteString(templateContent)
	if templateContent == "" {
		sb.WriteString("(no template content)")
	}
	sb.WriteString("\n\n## Output\n\n")
	sb.WriteString(fmt.Sprintf("- Per-source analysis: `%s`\n\n", outputFile))
	sb.WriteString("Work thoroughly. This is a comparative architecture study, not a surface skim.\n")
	return sb.String()
}

func BuildSynthesisPrompt(root string, dim types.Dimension, allSources []types.Source) string {
	dimFile := filepath.Join(root, "dimensions", dim.File)
	templateFile := filepath.Join(root, "templates", "report.md")
	synthFile := filepath.Join(root, "prompts", "synthesize.md")
	reportFile := fmt.Sprintf("reports/final/%s-%s.md", dim.Number, dim.Name)

	synthContent := readFile(synthFile)
	dimContent := readFile(dimFile)
	templateContent := readFile(templateFile)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Synthesis: %s\n\n", dim.Title))
	sb.WriteString("Read all per-source analysis files and create a combined study report.\n\n")
	sb.WriteString("## Synthesis Instructions\n\n")
	sb.WriteString(synthContent)
	if synthContent == "" {
		sb.WriteString("(no synthesis instructions)")
	}
	sb.WriteString("\n\n## Study Dimension\n\n")
	sb.WriteString(dimContent)
	if dimContent == "" {
		sb.WriteString("(no dimension content)")
	}
	sb.WriteString("\n\n## Sources Studied\n\n")
	for _, s := range allSources {
		sb.WriteString(fmt.Sprintf("- **%s**\n", s.Name))
	}
	sb.WriteString("\n## Per-Source Analysis Files to Read\n\n")
	for _, s := range allSources {
		sb.WriteString(fmt.Sprintf("   - `reports/source/%s-%s/%s.md`\n", dim.Number, dim.Name, s.Name))
	}
	sb.WriteString("\n## Instructions\n\n")
	sb.WriteString("1. Read ALL per-source analysis files listed above.\n")
	sb.WriteString("2. Follow the Synthesis Instructions and Study Dimension above.\n")
	sb.WriteString(fmt.Sprintf("3. Write the report to `%s` using the Report Template below.\n", reportFile))
	sb.WriteString("4. Fill in all template sections including cross-source comparison, synthesis, tradeoff matrix, and evidence index.\n")
	sb.WriteString("5. Do NOT access any source code directly — all evidence is already captured in the analysis files.\n\n")
	sb.WriteString("## Report Template\n\n")
	sb.WriteString(templateContent)
	if templateContent == "" {
		sb.WriteString("(no template content)")
	}
	sb.WriteString("\n\n## Output\n\n")
	sb.WriteString(fmt.Sprintf("- Combined report: `%s`\n\n", reportFile))
	sb.WriteString("Work thoroughly. This is a comparative architecture study, not a surface skim.\n")
	return sb.String()
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}