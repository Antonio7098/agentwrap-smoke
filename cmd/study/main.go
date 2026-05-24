package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Antonio7098/agentwrap"
	"github.com/Antonio7098/agentwrap/opencode"
	"github.com/spf13/cobra"
	"github.com/ultraplan/agentwrap-smoke/internal/code"
	"github.com/ultraplan/agentwrap-smoke/internal/config"
	"github.com/ultraplan/agentwrap-smoke/internal/discover"
	"github.com/ultraplan/agentwrap-smoke/internal/evolve"
	initpkg "github.com/ultraplan/agentwrap-smoke/internal/init"
	"github.com/ultraplan/agentwrap-smoke/internal/state"
	"github.com/ultraplan/agentwrap-smoke/internal/types"
)

var UltraPlanRoot = func() string {
	if envRoot := os.Getenv("ULTRAPLAN_ROOT"); envRoot != "" {
		return envRoot
	}
	cwd, _ := os.Getwd()
	return cwd
}()

func main() {
	root := &cobra.Command{
		Use:   "study",
		Short: "agentwrap-smoke study CLI",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	root.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available studies",
		Run:   listStudies,
	})

	codeCmd := &cobra.Command{
		Use:   "code",
		Short: "Extract code references from reports",
		Args:  cobra.MinimumNArgs(0),
		Run:   cmdCode,
	}
	codeCmd.Flags().String("output", "", "Output file")
	root.AddCommand(codeCmd)

	evolveCmd := &cobra.Command{
		Use:   "evolve",
		Short: "Trace evidence packs through reports",
		Args:  cobra.MinimumNArgs(0),
		Run:   cmdEvolve,
	}
	evolveCmd.Flags().Int("top-sources", 5, "Include top N per-source reports by score")
	evolveCmd.Flags().String("output", "", "Write output to file")
	evolveCmd.Flags().Bool("no-code", false, "Skip code extraction")
	evolveCmd.Flags().Bool("final-only", false, "Final reports only")
	root.AddCommand(evolveCmd)

	initCmd := &cobra.Command{
		Use:   "initialise-study",
		Short: "Create a new study from YAML config",
		Args:  cobra.ExactArgs(1),
		Run:   cmdInitialiseStudy,
	}
	initCmd.Flags().String("name", "", "Study name")
	initCmd.Flags().Bool("dry-run", false, "Dry run")
	initCmd.Flags().Bool("force", false, "Overwrite existing")
	initCmd.Flags().Bool("no-clone", false, "Skip cloning repos")
	initCmd.Flags().String("output-dir", "", "Output directory")
	root.AddCommand(initCmd)

	sprintPlanCmd := &cobra.Command{
		Use:   "sprint-plan",
		Short: "Plan a sprint for a target",
		Args:  cobra.ExactArgs(2),
		Run:   cmdSprintPlan,
	}
	sprintPlanCmd.Flags().String("model", "", "Model to use")
	sprintPlanCmd.Flags().String("variant", "", "Model variant")
	sprintPlanCmd.Flags().Bool("dry-run", false, "Dry run")
	root.AddCommand(sprintPlanCmd)

	execSprintCmd := &cobra.Command{
		Use:   "execute-sprint",
		Short: "Execute a planned sprint",
		Args:  cobra.ExactArgs(2),
		Run:   cmdExecuteSprint,
	}
	execSprintCmd.Flags().String("model", "", "Model to use")
	execSprintCmd.Flags().Bool("dry-run", false, "Dry run")
	root.AddCommand(execSprintCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func listStudies(cmd *cobra.Command, args []string) {
	studiesDir := filepath.Join(UltraPlanRoot, "studies")
	entries, err := os.ReadDir(studiesDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "No studies found.")
		return
	}
	fmt.Println()
	fmt.Println("Available Studies:")
	fmt.Println()
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			fmt.Printf("  %s\n", e.Name())
		}
	}
	fmt.Println("\nUsage: study <study-name> <command> [args]")
	fmt.Println("       study list")
	fmt.Println("       study code [--output <file>] <@report-file>...")
	fmt.Println("       study evolve [--top-sources <N>] [--output <file>] [--no-code] [--final-only] <@evidence-report>...")
	fmt.Println("       study initialise-study <study-init.yml> [options]")
	fmt.Println("       study sprint-plan <target> <sprint-slug> [options]")
	fmt.Println("       study execute-sprint <target> <sprint-slug> [options]")
	fmt.Println("")
}

func cmdCode(cmd *cobra.Command, args []string) {
	outputIdx := -1
	for i, a := range args {
		if a == "--output" && i+1 < len(args) {
			outputIdx = i + 1
			break
		}
	}
	var outputFile string
	fileArgs := args
	if outputIdx >= 0 {
		outputFile = args[outputIdx]
		fileArgs = append(args[:outputIdx-1], args[outputIdx+1:]...)
	}

	if len(fileArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no report files specified.\nUsage: study code [--output <file>] <@report-file>...")
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
			fmt.Fprintf(os.Stderr, "Error: report file not found: %s\n", p)
			os.Exit(1)
		}
	}

	output := code.ProcessReports(resolvedPaths)
	if outputFile != "" {
		os.WriteFile(outputFile, []byte(output), 0644)
		fmt.Printf("Code references written to %s\n", outputFile)
	} else {
		fmt.Print(output)
	}
}

func cmdEvolve(cmd *cobra.Command, args []string) {
	opts := evolve.EvolveOptions{}
	if topSources, _ := cmd.Flags().GetInt("top-sources"); topSources > 0 {
		opts.TopSources = topSources
	}
	if output, _ := cmd.Flags().GetString("output"); output != "" {
		opts.OutputFile = output
	}
	if noCode, _ := cmd.Flags().GetBool("no-code"); noCode {
		opts.NoCode = true
	}
	if finalOnly, _ := cmd.Flags().GetBool("final-only"); finalOnly {
		opts.FinalOnly = true
	}
	evolve.CmdEvolve(args, opts)
}

func cmdInitialiseStudy(cmd *cobra.Command, args []string) {
	opts := struct {
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
	}{}
	opts.Name, _ = cmd.Flags().GetString("name")
	opts.DryRun, _ = cmd.Flags().GetBool("dry-run")
	opts.Force, _ = cmd.Flags().GetBool("force")
	opts.NoClone, _ = cmd.Flags().GetBool("no-clone")
	opts.OutputDir, _ = cmd.Flags().GetString("output-dir")
	initpkg.CmdInitialiseStudy(args[0], opts)
}

func cmdSprintPlan(cmd *cobra.Command, args []string) {
	target := args[0]
	sprintSlug := args[1]
	cfg, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	model, _ := cmd.Flags().GetString("model")
	variant, _ := cmd.Flags().GetString("variant")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if model == "" {
		model = cfg.SprintPlanningModel
	}
	if variant == "" {
		variant = cfg.DefaultVariant
	}

	targetDir := filepath.Join(UltraPlanRoot, "targets", target)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Target %q not found at targets/%s\n", target, target)
		os.Exit(1)
	}

	promptPath := filepath.Join(UltraPlanRoot, "prompts", "plan-sprint.md")
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Sprint planning prompt not found at %s\n", promptPath)
		os.Exit(1)
	}

	promptContent, _ := os.ReadFile(promptPath)
	processPrompt := strings.ReplaceAll(string(promptContent), "{target}", target)
	processPrompt = strings.ReplaceAll(processPrompt, "{sprint-slug}", sprintSlug)

	outputDir := filepath.Join(targetDir, "sprints", sprintSlug)
	outputFile := filepath.Join(outputDir, "plan.md")

	if dryRun {
		fmt.Printf("\n=== DRY RUN: plan-sprint %s %s ===\n\n", target, sprintSlug)
		fmt.Printf("Prompt file: %s\n", promptPath)
		fmt.Printf("Output file: %s\n", outputFile)
		fmt.Printf("Model: %s\n", model)
		fmt.Println("")
		return
	}

	os.MkdirAll(outputDir, 0755)
	fmt.Printf("\n▶ Planning sprint %s for target %s...\n\n", sprintSlug, target)

	runtime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + filepath.Join(UltraPlanRoot, "opencode-config.json")),
	)

	run, err := runtime.StartRun(cmd.Context(), agentwrap.RunRequest{
		Prompt:   processPrompt,
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  time.Duration(cfg.DefaultTimeoutMs) * time.Millisecond,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting run: %v\n", err)
		os.Exit(1)
	}

	result, err := run.Wait(cmd.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error waiting for run: %v\n", err)
		os.Exit(1)
	}

	if result.Status == agentwrap.StatusCompleted {
		fmt.Printf("\n✓ Sprint plan written: %s\n", outputFile)
	} else {
		fmt.Fprintf(os.Stderr, "\n✗ Sprint planning failed (exit code unknown)\n")
		os.Exit(1)
	}
}

func cmdExecuteSprint(cmd *cobra.Command, args []string) {
	target := args[0]
	sprintSlug := args[1]
	cfg, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	model, _ := cmd.Flags().GetString("model")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if model == "" {
		model = cfg.SprintExecutionModel
	}

	targetDir := filepath.Join(UltraPlanRoot, "targets", target)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Target %q not found at targets/%s\n", target, target)
		os.Exit(1)
	}

	sprintDir := filepath.Join(targetDir, "sprints", sprintSlug)
	planPath := filepath.Join(sprintDir, "plan.md")
	reasoningPath := filepath.Join(sprintDir, "reasoning.md")

	missing := []string{}
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		missing = append(missing, planPath)
	}
	if _, err := os.Stat(reasoningPath); os.IsNotExist(err) {
		missing = append(missing, reasoningPath)
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "\nError: Cannot execute sprint %q for target %q. Missing required planning artefacts:\n", sprintSlug, target)
		for _, p := range missing {
			fmt.Fprintf(os.Stderr, "  ✗ %s\n", p)
		}
		fmt.Fprintf(os.Stderr, "\n  Run 'study sprint-plan %s %s' to generate the missing artefacts.\n\n", target, sprintSlug)
		os.Exit(1)
	}

	promptPath := filepath.Join(UltraPlanRoot, "prompts", "execute-sprint.md")
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Sprint execution prompt not found at %s\n", promptPath)
		os.Exit(1)
	}

	promptContent, _ := os.ReadFile(promptPath)
	processPrompt := strings.ReplaceAll(string(promptContent), "{target}", target)
	processPrompt = strings.ReplaceAll(processPrompt, "{sprint-slug}", sprintSlug)

	if dryRun {
		fmt.Printf("\n=== DRY RUN: execute-sprint %s %s ===\n\n", target, sprintSlug)
		fmt.Printf("Prompt file: %s\n", promptPath)
		fmt.Printf("Model: %s\n", model)
		fmt.Println("")
		return
	}

	fmt.Printf("\n▶ Executing sprint %s for target %s...\n\n", sprintSlug, target)

	runtime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + filepath.Join(UltraPlanRoot, "opencode-config.json")),
	)

	run, err := runtime.StartRun(cmd.Context(), agentwrap.RunRequest{
		Prompt:   processPrompt,
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  time.Duration(cfg.DefaultTimeoutMs) * time.Millisecond,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting run: %v\n", err)
		os.Exit(1)
	}

	result, err := run.Wait(cmd.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error waiting for run: %v\n", err)
		os.Exit(1)
	}

	if result.Status == agentwrap.StatusCompleted {
		fmt.Printf("\n✓ Sprint execution complete: %s\n", planPath)
	} else {
		fmt.Fprintf(os.Stderr, "\n✗ Sprint execution failed\n")
		os.Exit(1)
	}
}

func gitCommitAndPush(msg string) {
	exec.Command("git", "add", "-A").Run()
	exec.Command("git", "commit", "-m", msg).Run()
	exec.Command("git", "push").Run()
}

func cmdListStudy(studyName string) {
	studyDir := filepath.Join(UltraPlanRoot, "studies", studyName)

	sources := discover.Sources(studyDir)
	dims := discover.Dimensions(studyDir)

	fmt.Println()
	fmt.Println("Available Sources:")
	fmt.Println()
	for _, s := range sources {
		fmt.Printf("  %s\n", s.Name)
	}

	fmt.Println()
	fmt.Println("Available Dimensions:")
	fmt.Println()
	for _, d := range dims {
		fmt.Printf("  %s-%s.md — %s\n", d.Number, d.Name, d.Title)
	}

	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Printf("  study %s run <dimension-ref> <source-name> [options]\n", studyName)
	fmt.Printf("  study %s run-all [options]\n", studyName)
	fmt.Printf("  study %s run-loop [options]\n", studyName)
	fmt.Printf("  study %s status\n", studyName)
	fmt.Printf("  study %s list\n\n", studyName)
}

func cmdStatus(studyName string) {
	studyDir := filepath.Join(UltraPlanRoot, "studies", studyName)

	st, err := state.Load(studyDir)
	if err != nil || st == nil {
		fmt.Println("\nNo run state found. Start a run with: study run-loop")
		return
	}

	analysisTotal := len(st.Tasks)
	analysisCompleted := 0
	analysisRunning := 0
	analysisFailed := 0
	analysisPending := 0
	for _, t := range st.Tasks {
		switch t.Status {
		case types.StatusCompleted:
			analysisCompleted++
		case types.StatusRunning:
			analysisRunning++
		case types.StatusFailed:
			analysisFailed++
		case types.StatusPending:
			analysisPending++
		}
	}

	synthTotal := len(st.SynthesisTasks)
	synthCompleted := 0
	synthRunning := 0
	synthFailed := 0
	synthPending := 0
	for _, s := range st.SynthesisTasks {
		switch s.Status {
		case types.StatusCompleted:
			synthCompleted++
		case types.StatusRunning:
			synthRunning++
		case types.StatusFailed:
			synthFailed++
		case types.StatusPending:
			synthPending++
		}
	}

	grandTotal := analysisTotal + synthTotal
	grandCompleted := analysisCompleted + synthCompleted
	dimCount := 0
	seenDims := make(map[string]bool)
	for _, t := range st.Tasks {
		if !seenDims[t.DimensionNumber] {
			seenDims[t.DimensionNumber] = true
			dimCount++
		}
	}

	fmt.Printf("\nStarted: %s\n", st.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Printf("Updated: %s\n", st.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if st.IsComplete {
		fmt.Printf("Status: ✓ Complete  |  Batch: %d\n", st.BatchSize)
	} else {
		fmt.Printf("Status: ▶ In progress  |  Batch: %d\n", st.BatchSize)
	}

	if analysisFailed > 0 || analysisPending > 0 || synthFailed > 0 || synthPending > 0 {
		fmt.Println("")
		fmt.Println("Remaining:")
		for _, t := range st.Tasks {
			if t.Status == types.StatusCompleted {
				continue
			}
			label := t.DimensionTitle + " × " + t.SourceName
			switch t.Status {
			case types.StatusRunning:
				fmt.Printf("  ▶ %s (attempt %d)\n", label, t.Attempts)
			case types.StatusFailed:
				retryStr := ""
				if t.NextRetryAt != nil {
					retryStr = ", retry at " + t.NextRetryAt.Format("2006-01-02T15:04:05Z07:00")
				}
				fmt.Printf("  ✗ %s (attempt %d%s)\n", label, t.Attempts, retryStr)
				if t.LastError != "" {
					fmt.Printf("    Error: %s\n", t.LastError)
				}
			default:
				fmt.Printf("  ○ %s\n", label)
			}
		}
		for _, s := range st.SynthesisTasks {
			if s.Status == types.StatusCompleted {
				continue
			}
			label := "Synthesis: " + s.DimensionTitle
			switch s.Status {
			case types.StatusRunning:
				fmt.Printf("  ▶ %s (attempt %d)\n", label, s.Attempts)
			case types.StatusFailed:
				retryStr := ""
				if s.NextRetryAt != nil {
					retryStr = ", retry at " + s.NextRetryAt.Format("2006-01-02T15:04:05Z07:00")
				}
				fmt.Printf("  ✗ %s (attempt %d%s)\n", label, s.Attempts, retryStr)
				if s.LastError != "" {
					fmt.Printf("    Error: %s\n", s.LastError)
				}
			default:
				fmt.Printf("  ○ %s\n", label)
			}
		}
		fmt.Println("")
	}

	fmt.Printf("Dimensions: %d  |  Analyses: %d/%d", dimCount, analysisCompleted, analysisTotal)
	if synthTotal > 0 {
		fmt.Printf("  |  Synthesis: %d/%d  |  Total: %d/%d", synthCompleted, synthTotal, grandCompleted, grandTotal)
	} else {
		fmt.Printf("  |  Total: %d/%d", grandCompleted, grandTotal)
	}
	fmt.Println()
	fmt.Println()
}
