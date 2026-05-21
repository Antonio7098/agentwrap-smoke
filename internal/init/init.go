package init

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type RepoItem struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

type DimensionItem struct {
	Number      string   `yaml:"number"`
	Name        string   `yaml:"name"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description,omitempty"`
	Purpose     string   `yaml:"purpose,omitempty"`
	Steps       []string `yaml:"steps,omitempty"`
	Evidence    []string `yaml:"evidence,omitempty"`
	Questions   []string `yaml:"questions,omitempty"`
}

type StudyInitConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Repos       struct {
		Count int        `yaml:"count"`
		Items []RepoItem `yaml:"items"`
	} `yaml:"repos"`
	Dimensions struct {
		Count int             `yaml:"count"`
		Items []DimensionItem `yaml:"items"`
	} `yaml:"dimensions"`
}

func CmdInitialiseStudy(yamlPath string, opts struct {
	Name       string
	Repos      int
	Dimensions int
	Model      string
	Variant    string
	DryRun     bool
	Force      bool
	NoClone    bool
	TimeoutMs  int64
	OutputDir  string
}) {
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", yamlPath)
		os.Exit(1)
	}

	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading YAML file: %v\n", err)
		os.Exit(1)
	}

	var config StudyInitConfig
	if err := yaml.Unmarshal(yamlContent, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML file: %v\n", err)
		os.Exit(1)
	}

	if opts.Name != "" {
		config.Name = opts.Name
	}
	if opts.Repos > 0 {
		config.Repos.Count = opts.Repos
	}
	if opts.Dimensions > 0 {
		config.Dimensions.Count = opts.Dimensions
	}

	for i := range config.Dimensions.Items {
		num := config.Dimensions.Items[i].Number
		if len(num) == 1 {
			num = "0" + num
			config.Dimensions.Items[i].Number = num
		}
	}

	if config.Name == "" {
		fmt.Fprintln(os.Stderr, "Error: study name is required (in YAML or via --name)")
		os.Exit(1)
	}
	if config.Repos.Count < 1 {
		fmt.Fprintln(os.Stderr, "Error: at least 1 repo required")
		os.Exit(1)
	}
	if config.Dimensions.Count < 1 {
		fmt.Fprintln(os.Stderr, "Error: at least 1 dimension required")
		os.Exit(1)
	}

	ultraPlanRoot := os.Getenv("ULTRAPLAN_ROOT")
	if ultraPlanRoot == "" {
		ultraPlanRoot, _ = os.Getwd()
	}

	studiesRoot := ultraPlanRoot
	if opts.OutputDir != "" {
		studiesRoot = filepath.Join(ultraPlanRoot, opts.OutputDir)
	}
	studyDir := filepath.Join(studiesRoot, config.Name)

	if !opts.Force && exists(studyDir) {
		fmt.Fprintf(os.Stderr, "Error: Study %q already exists at %s\n", config.Name, studyDir)
		fmt.Fprintln(os.Stderr, "  Use --force to overwrite existing study directory")
		os.Exit(1)
	}

	if opts.DryRun {
		fmt.Println()
		fmt.Println("=== DRY RUN: initialise-study ===")
		fmt.Println()
		fmt.Printf("YAML file: %s\n", yamlPath)
		fmt.Printf("Study name: %s\n", config.Name)
		fmt.Printf("Description: %s\n", desc(config.Description))
		fmt.Printf("Output dir: %s\n", studyDir)
		fmt.Printf("Repos:  %d defined, targeting %d\n", len(config.Repos.Items), config.Repos.Count)
		fmt.Printf("Dims:   %d defined, targeting %d\n", len(config.Dimensions.Items), config.Dimensions.Count)
		reposShort := config.Repos.Count - len(config.Repos.Items)
		dimsShort := config.Dimensions.Count - len(config.Dimensions.Items)
		if reposShort > 0 {
			fmt.Printf("\nWould research %d additional repos via OpenCode\n", reposShort)
		}
		if dimsShort > 0 {
			fmt.Printf("Would research %d additional dimensions via OpenCode\n", dimsShort)
		}
		fmt.Println()
		fmt.Println("Would create:")
		fmt.Println()
		fmt.Printf("  %s/\n", studyDir)
		fmt.Printf("  %s/dimensions/  (%d .md files)\n", studyDir, config.Dimensions.Count)
		fmt.Printf("  %s/sources/     (%d repos → git clone --depth 1)\n", studyDir, config.Repos.Count)
		fmt.Printf("  %s/reports/source/\n", studyDir)
		fmt.Printf("  %s/reports/final/\n", studyDir)
		fmt.Printf("  %s/study-init.yml\n", studyDir)
		fmt.Printf("  %s/README.md\n", studyDir)
		if opts.NoClone {
			fmt.Println("\n  (skipping clone: --no-clone)")
		}
		fmt.Println("")
		return
	}

	fmt.Printf("\n▶ Initialising study: %s\n", config.Name)
	if config.Description != "" {
		fmt.Printf("  Description: %s\n", config.Description)
	}

	reposShort := config.Repos.Count - len(config.Repos.Items)
	dimsShort := config.Dimensions.Count - len(config.Dimensions.Items)

	// reposShort and dimsShort represent how many additional repos/dims
	// need to be researched. Currently research via OpenCode is not implemented.
	_ = reposShort
	_ = dimsShort

	if opts.Force && exists(studyDir) {
		fmt.Printf("  ⚠ Removing existing study directory (--force)\n")
		os.RemoveAll(studyDir)
	}
	os.MkdirAll(filepath.Join(studyDir, "dimensions"), 0755)
	os.MkdirAll(filepath.Join(studyDir, "sources"), 0755)
	os.MkdirAll(filepath.Join(studyDir, "reports", "source"), 0755)
	os.MkdirAll(filepath.Join(studyDir, "reports", "final"), 0755)

	sortedDims := make([]DimensionItem, len(config.Dimensions.Items))
	copy(sortedDims, config.Dimensions.Items)
	for i := 0; i < len(sortedDims)-1; i++ {
		for j := i + 1; j < len(sortedDims); j++ {
			if strings.Compare(sortedDims[i].Number, sortedDims[j].Number) > 0 {
				sortedDims[i], sortedDims[j] = sortedDims[j], sortedDims[i]
			}
		}
	}

	for _, dim := range sortedDims {
		content := generateDimensionMarkdown(dim)
		dimFile := fmt.Sprintf("%s-%s.md", dim.Number, dim.Name)
		os.WriteFile(filepath.Join(studyDir, "dimensions", dimFile), []byte(content), 0644)
		fmt.Printf("  ✓ Dimension: %s\n", dimFile)
	}

	fullYaml, _ := yaml.Marshal(config)
	os.WriteFile(filepath.Join(studyDir, "study-init.yml"), fullYaml, 0644)
	fmt.Printf("  ✓ study-init.yml written\n")

	var repoRows, dimRows []string
	for _, r := range config.Repos.Items {
		repoRows = append(repoRows, fmt.Sprintf("| %s | `%s` | %s |", r.Name, r.URL, desc(r.Description)))
	}
	for _, d := range sortedDims {
		dimRows = append(dimRows, fmt.Sprintf("| %s | %s | %s |", d.Number, d.Title, desc(d.Description)))
	}

	studyReadme := "# " + config.Name + "\n\n" + desc(config.Description) + "\n\n## Repositories Studied\n\n| Name | URL | Description |\n|------|-----|-------------|\n" + strings.Join(repoRows, "\n") + "\n\n## Study Dimensions\n\n| # | Dimension | Description |\n|---|-----------|-------------|\n" + strings.Join(dimRows, "\n") + "\n\n## Usage\n\n```bash\n# List sources and dimensions\nstudy " + config.Name + " list\n\n# Run all dimension × source analyses\nstudy " + config.Name + " run-all --parallel 3\n\n# Stateful batch runner with retry/backoff\nstudy " + config.Name + " run-loop --batch-size 2\n\n# Show run-loop status\nstudy " + config.Name + " status\n```\n"

	os.WriteFile(filepath.Join(studyDir, "README.md"), []byte(studyReadme), 0644)
	fmt.Printf("  ✓ README.md written\n")

	if !opts.NoClone {
		fmt.Printf("\n▶ Cloning repos into sources/...\n\n")
		sourcesDir := filepath.Join(studyDir, "sources")
		cloned, failed := 0, 0
		for _, repo := range config.Repos.Items {
			dest := filepath.Join(sourcesDir, repo.Name)
			if exists(dest) {
				fmt.Printf("  ○ %s already exists, skipping\n", repo.Name)
				cloned++
				continue
			}
			fmt.Printf("  ○ Cloning %s...\n", repo.Name)
			cmd := exec.Command("git", "clone", "--depth", "1", repo.URL, dest)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ %s failed: %v\n", repo.Name, err)
				failed++
			} else {
				fmt.Printf("  ✓ %s cloned\n", repo.Name)
				cloned++
			}
		}
		fmt.Printf("\n  Cloned: %d, failed: %d\n", cloned, failed)
	}

	fmt.Printf("\n✓ Study %q initialised at %s\n", config.Name, studyDir)
	fmt.Printf("  Dimensions: %d\n", len(config.Dimensions.Items))
	fmt.Printf("  Repos:      %d\n", len(config.Repos.Items))
	fmt.Println("")

	if len(config.Dimensions.Items) < config.Dimensions.Count {
		fmt.Printf("  ⚠ Dimensions: %d found, target was %d\n", len(config.Dimensions.Items), config.Dimensions.Count)
		fmt.Printf("     Edit study-init.yml and re-run or add dimension files manually.\n")
	}
	if len(config.Repos.Items) < config.Repos.Count {
		fmt.Printf("  ⚠ Repos: %d found, target was %d\n", len(config.Repos.Items), config.Repos.Count)
		fmt.Printf("     Edit study-init.yml and re-run or populate sources/ manually.\n")
	}

	fmt.Println("\nNext steps:")
	if opts.NoClone {
		fmt.Println("  1. Populate sources/ with git clones:")
		fmt.Printf("     git clone <url> %s\n", filepath.Join(studyDir, "sources"))
		fmt.Printf("  2. Review and edit dimension files in %s\n", filepath.Join(studyDir, "dimensions"))
		fmt.Printf("  3. Run: study %s run-all\n", config.Name)
	} else {
		fmt.Printf("  1. Review and edit dimension files in %s\n", filepath.Join(studyDir, "dimensions"))
		fmt.Printf("  2. Run: study %s run-all\n", config.Name)
	}
	fmt.Println("")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func desc(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func generateDimensionMarkdown(dim DimensionItem) string {
	purpose := dim.Purpose
	if purpose == "" {
		purpose = fmt.Sprintf("Analysis of %s across source projects.", strings.ToLower(dim.Title))
	}
	steps := dim.Steps
	if len(steps) == 0 {
		steps = []string{
			"Read prompts/base.md for execution instructions.",
			"For the target repo: identify how " + dim.Name + " is approached.",
			"Answer the questions below and collect evidence.",
		}
	}
	evidence := dim.Evidence
	if len(evidence) == 0 {
		evidence = []string{
			fmt.Sprintf("Source files implementing %s patterns", dim.Name),
			"Configuration and type definitions",
			"Tests encoding expected behavior",
		}
	}
	questions := dim.Questions
	if len(questions) == 0 {
		questions = []string{
			fmt.Sprintf("How does each source implement %s?", strings.ToLower(dim.Title)),
			"What are the key differences between approaches?",
			"What tradeoffs are visible in each implementation?",
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Dimension: %s\n\n", dim.Title))
	sb.WriteString("## Purpose\n\n")
	sb.WriteString(purpose + "\n\n")
	sb.WriteString("## Steps\n\n")
	for i, s := range steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
	}
	sb.WriteString("\n## Evidence\n\n")
	for _, e := range evidence {
		sb.WriteString(fmt.Sprintf("- %s\n", e))
	}
	sb.WriteString("\n## Questions\n\n")
	for i, q := range questions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, q))
	}
	sb.WriteString("\n## Rating\n\n")
	sb.WriteString("Assign a score from 1-10 based on the analysis findings.\n\n")
	sb.WriteString("| Score | Meaning |\n")
	sb.WriteString("| ----- | ------- |\n")
	sb.WriteString("| 1-3 | Poor implementation or absent |\n")
	sb.WriteString("| 4-6 | Basic implementation with gaps |\n")
	sb.WriteString("| 7-8 | Good implementation with minor issues |\n")
	sb.WriteString("| 9-10 | Excellent, exemplar implementation |\n\n")
	sb.WriteString("## Output\n\n")
	sb.WriteString(fmt.Sprintf("Write findings to `reports/source/{NN}-%s/{source-name}.md` using `../../templates/repo-analysis.md`.\n", dim.Name))
	return sb.String()
}
