package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Antonio7098/agentwrap"
	"github.com/Antonio7098/agentwrap/opencode"
	"github.com/spf13/cobra"
	"github.com/ultraplan/agentwrap-smoke/internal/config"
)

var UltraPlanRoot = func() string {
	if envRoot := os.Getenv("ULTRAPLAN_ROOT"); envRoot != "" {
		return envRoot
	}
	cwd, _ := os.Getwd()
	return cwd
}()

type runConfig struct {
	studyName      string
	dimension      string
	source         string
	model          string
	variant        string
	sessionID      string
	sessionCont    bool
	maxAttempts    int
	timeoutMs      int64
	validate       bool
	observe        bool
	healthCheck    bool
	logDir         string
	expectStatus   string
	expectCategory string
	dbExecutable   string
	dbEnv          []string
}

type scenarioExpectation struct {
	Status   string `json:"status,omitempty"`
	Category string `json:"category,omitempty"`
}

type scenarioResult struct {
	Name      string               `json:"name"`
	Passed    bool                 `json:"passed"`
	Status    string               `json:"status"`
	Category  string               `json:"category,omitempty"`
	SessionID string               `json:"session_id,omitempty"`
	Err       string               `json:"error,omitempty"`
	LogDir    string               `json:"log_dir"`
	Warnings  []string             `json:"warnings,omitempty"`
	Usage     *agentwrap.Usage     `json:"usage,omitempty"`
	Expect    *scenarioExpectation `json:"expect,omitempty"`
}

func main() {
	root := &cobra.Command{
		Use:   "agentwrap-run",
		Short: "Run studies with full agentwrap instrumentation",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run a study dimension on a source with agentwrap",
		Args:  cobra.ExactArgs(3),
		Run:   cmdRun,
	}
	runCmd.Flags().String("study", "", "Study name")
	runCmd.Flags().String("dimension", "", "Dimension (e.g., 01-project-structure)")
	runCmd.Flags().String("source", "", "Source name (e.g., age)")
	runCmd.Flags().String("model", "", "Model to use")
	runCmd.Flags().String("variant", "", "Model variant")
	runCmd.Flags().String("session", "", "Session ID for continuation")
	runCmd.Flags().Bool("session-continue", false, "Continue previous session")
	runCmd.Flags().Int("max-attempts", 3, "Max retry attempts")
	runCmd.Flags().Bool("validate", true, "Enable output validation")
	runCmd.Flags().Bool("observe", true, "Enable observability")
	runCmd.Flags().Bool("health-check", true, "Run health checks pre-flight")
	runCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(runCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available studies, dimensions, and sources",
		Args:  cobra.ExactArgs(1),
		Run:   cmdList,
	}
	root.AddCommand(listCmd)

	sessionCmd := &cobra.Command{
		Use:   "session-continue",
		Short: "Continue a previous session",
		Args:  cobra.ExactArgs(2),
		Run:   cmdSessionContinue,
	}
	sessionCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(sessionCmd)

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Test run cancellation",
		Args:  cobra.ExactArgs(0),
		Run:   cmdCancel,
	}
	cancelCmd.Flags().String("log-dir", "", "Log output directory")
	cancelCmd.Flags().String("model", "", "Model to use")
	cancelCmd.Flags().String("expect-status", "cancelled", "Expected final status")
	cancelCmd.Flags().String("expect-category", "cancellation", "Expected error category")
	root.AddCommand(cancelCmd)

	timeoutCmd := &cobra.Command{
		Use:   "timeout",
		Short: "Test run timeout",
		Args:  cobra.ExactArgs(0),
		Run:   cmdTimeout,
	}
	timeoutCmd.Flags().String("log-dir", "", "Log output directory")
	timeoutCmd.Flags().String("model", "", "Model to use")
	timeoutCmd.Flags().Int64("timeout-ms", 500, "Timeout in milliseconds")
	timeoutCmd.Flags().String("expect-status", "failed", "Expected final status")
	timeoutCmd.Flags().String("expect-category", "timeout", "Expected error category")
	root.AddCommand(timeoutCmd)

	rateLimitCmd := &cobra.Command{
		Use:   "rate-limit",
		Short: "Test rate limit fallback",
		Args:  cobra.ExactArgs(0),
		Run:   cmdRateLimit,
	}
	rateLimitCmd.Flags().String("log-dir", "", "Log output directory")
	rateLimitCmd.Flags().Bool("use-fake", false, "Use fake-opencode with rate_limit_nested mode (deterministic fixture)")
	rateLimitCmd.Flags().String("primary-model", "opencode/gpt-5.5", "Primary model expected to fail")
	rateLimitCmd.Flags().String("fallback-model", "opencode/deepseek-v4-flash-free", "Fallback model to use")
	root.AddCommand(rateLimitCmd)

	fixedBackoffCmd := &cobra.Command{
		Use:   "fixed-backoff",
		Short: "Test FixedBackoff policy",
		Args:  cobra.ExactArgs(0),
		Run:   cmdFixedBackoff,
	}
	fixedBackoffCmd.Flags().String("log-dir", "", "Log output directory")
	fixedBackoffCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(fixedBackoffCmd)

	usageCmd := &cobra.Command{
		Use:   "usage",
		Short: "Test usage tracking",
		Args:  cobra.ExactArgs(0),
		Run:   cmdUsage,
	}
	usageCmd.Flags().String("log-dir", "", "Log output directory")
	usageCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(usageCmd)

	artifactsCmd := &cobra.Command{
		Use:   "artifacts",
		Short: "Test artifacts projection",
		Args:  cobra.ExactArgs(0),
		Run:   cmdArtifacts,
	}
	artifactsCmd.Flags().String("log-dir", "", "Log output directory")
	artifactsCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(artifactsCmd)

	validateJsonCmd := &cobra.Command{
		Use:   "validate-json",
		Short: "Test JSON validation",
		Args:  cobra.ExactArgs(0),
		Run:   cmdValidateJson,
	}
	validateJsonCmd.Flags().String("log-dir", "", "Log output directory")
	validateJsonCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(validateJsonCmd)

	validateMdCmd := &cobra.Command{
		Use:   "validate-md",
		Short: "Test MarkdownTemplate validation",
		Args:  cobra.ExactArgs(0),
		Run:   cmdValidateMd,
	}
	validateMdCmd.Flags().String("log-dir", "", "Log output directory")
	validateMdCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(validateMdCmd)

	customValidatorCmd := &cobra.Command{
		Use:   "custom-validator",
		Short: "Test custom validator",
		Args:  cobra.ExactArgs(0),
		Run:   cmdCustomValidator,
	}
	customValidatorCmd.Flags().String("log-dir", "", "Log output directory")
	customValidatorCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(customValidatorCmd)

	healthFailCmd := &cobra.Command{
		Use:   "health-fail",
		Short: "Test health check failures",
		Args:  cobra.ExactArgs(0),
		Run:   cmdHealthFail,
	}
	healthFailCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(healthFailCmd)

	sessionForkCmd := &cobra.Command{
		Use:   "session-fork",
		Short: "Test session fork (unsupported)",
		Args:  cobra.ExactArgs(0),
		Run:   cmdSessionFork,
	}
	sessionForkCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(sessionForkCmd)

	smokeTextCmd := &cobra.Command{
		Use:   "smoke-text",
		Short: "Test shortest successful completion",
		Args:  cobra.ExactArgs(0),
		Run:   cmdSmokeText,
	}
	smokeTextCmd.Flags().String("log-dir", "", "Log output directory")
	smokeTextCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(smokeTextCmd)

	smokeReasoningCmd := &cobra.Command{
		Use:   "smoke-reasoning",
		Short: "Test reasoning/text completion",
		Args:  cobra.ExactArgs(0),
		Run:   cmdSmokeReasoning,
	}
	smokeReasoningCmd.Flags().String("log-dir", "", "Log output directory")
	smokeReasoningCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(smokeReasoningCmd)

	smokeFileWriteCmd := &cobra.Command{
		Use:   "smoke-file-write",
		Short: "Test file write completion",
		Args:  cobra.ExactArgs(0),
		Run:   cmdSmokeFileWrite,
	}
	smokeFileWriteCmd.Flags().String("log-dir", "", "Log output directory")
	smokeFileWriteCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(smokeFileWriteCmd)

	validateFailCmd := &cobra.Command{
		Use:   "validate-fail",
		Short: "Test required validation failure",
		Args:  cobra.ExactArgs(0),
		Run:   cmdValidateFail,
	}
	validateFailCmd.Flags().String("log-dir", "", "Log output directory")
	validateFailCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(validateFailCmd)

	validateRepairCmd := &cobra.Command{
		Use:   "validate-repair",
		Short: "Test validation repair succeeds",
		Args:  cobra.ExactArgs(0),
		Run:   cmdValidateRepair,
	}
	validateRepairCmd.Flags().String("log-dir", "", "Log output directory")
	validateRepairCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(validateRepairCmd)

	validateRepairExhaustCmd := &cobra.Command{
		Use:   "validate-repair-exhaust",
		Short: "Test validation repair exhaustion",
		Args:  cobra.ExactArgs(0),
		Run:   cmdValidateRepairExhaust,
	}
	validateRepairExhaustCmd.Flags().String("log-dir", "", "Log output directory")
	validateRepairExhaustCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(validateRepairExhaustCmd)

	fallbackInvalidModelCmd := &cobra.Command{
		Use:   "fallback-invalid-model",
		Short: "Test fallback when primary model is invalid",
		Args:  cobra.ExactArgs(0),
		Run:   cmdFallbackInvalidModel,
	}
	fallbackInvalidModelCmd.Flags().String("log-dir", "", "Log output directory")
	fallbackInvalidModelCmd.Flags().String("primary-model", "opencode/not-a-real-model", "Invalid primary model")
	fallbackInvalidModelCmd.Flags().String("fallback-model", "opencode/deepseek-v4-flash-free", "Fallback model")
	root.AddCommand(fallbackInvalidModelCmd)

	fallbackInvalidProviderCmd := &cobra.Command{
		Use:   "fallback-invalid-provider",
		Short: "Test fallback when primary provider is invalid",
		Args:  cobra.ExactArgs(0),
		Run:   cmdFallbackInvalidProvider,
	}
	fallbackInvalidProviderCmd.Flags().String("log-dir", "", "Log output directory")
	fallbackInvalidProviderCmd.Flags().String("primary-model", "not-a-provider/model", "Invalid primary model/provider")
	fallbackInvalidProviderCmd.Flags().String("fallback-model", "opencode/deepseek-v4-flash-free", "Fallback model")
	root.AddCommand(fallbackInvalidProviderCmd)

	invalidSeparateProviderCmd := &cobra.Command{
		Use:   "invalid-separate-provider",
		Short: "Test wrapper provider syntax pre-validation",
		Args:  cobra.ExactArgs(0),
		Run:   cmdInvalidSeparateProvider,
	}
	invalidSeparateProviderCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(invalidSeparateProviderCmd)

	sessionFreshCmd := &cobra.Command{
		Use:   "session-fresh",
		Short: "Test fresh session success",
		Args:  cobra.ExactArgs(0),
		Run:   cmdSessionFresh,
	}
	sessionFreshCmd.Flags().String("log-dir", "", "Log output directory")
	sessionFreshCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(sessionFreshCmd)

	sessionContinueExistingCmd := &cobra.Command{
		Use:   "session-continue-existing",
		Short: "Test continue existing session success",
		Args:  cobra.ExactArgs(0),
		Run:   cmdSessionContinueExisting,
	}
	sessionContinueExistingCmd.Flags().String("log-dir", "", "Log output directory")
	sessionContinueExistingCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(sessionContinueExistingCmd)

	sessionContinueMissingCmd := &cobra.Command{
		Use:   "session-continue-missing",
		Short: "Test continue with missing/invalid session ID",
		Args:  cobra.ExactArgs(0),
		Run:   cmdSessionContinueMissing,
	}
	sessionContinueMissingCmd.Flags().String("log-dir", "", "Log output directory")
	sessionContinueMissingCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(sessionContinueMissingCmd)

	sessionContinueAfterFailCmd := &cobra.Command{
		Use:   "session-continue-after-fail",
		Short: "Test continue session after failed run",
		Args:  cobra.ExactArgs(0),
		Run:   cmdSessionContinueAfterFail,
	}
	sessionContinueAfterFailCmd.Flags().String("log-dir", "", "Log output directory")
	sessionContinueAfterFailCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(sessionContinueAfterFailCmd)

	repairWithContinueCmd := &cobra.Command{
		Use:   "repair-with-continue",
		Short: "Test repair with SessionActionContinue",
		Args:  cobra.ExactArgs(0),
		Run:   cmdRepairWithContinue,
	}
	repairWithContinueCmd.Flags().String("log-dir", "", "Log output directory")
	repairWithContinueCmd.Flags().String("model", "", "Model to use")
	root.AddCommand(repairWithContinueCmd)

	fallbackAllFailCmd := &cobra.Command{
		Use:   "fallback-all-fail",
		Short: "Test when both primary and fallback fail",
		Args:  cobra.ExactArgs(0),
		Run:   cmdFallbackAllFail,
	}
	fallbackAllFailCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(fallbackAllFailCmd)

	// Workstream 7: Process-Group Cleanup and Final-State Precedence smoke tests
	processGroupNonZeroFinalCmd := &cobra.Command{
		Use:   "process-group-nonzero-final",
		Short: "Test non-zero exit with final event (expected: completed)",
		Args:  cobra.ExactArgs(0),
		Run:   cmdProcessGroupNonZeroFinal,
	}
	processGroupNonZeroFinalCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(processGroupNonZeroFinalCmd)

	processGroupNonZeroRateLimitCmd := &cobra.Command{
		Use:   "process-group-nonzero-ratelimit",
		Short: "Test non-zero exit with rate-limit stderr (expected: rate_limit)",
		Args:  cobra.ExactArgs(0),
		Run:   cmdProcessGroupNonZeroRateLimit,
	}
	processGroupNonZeroRateLimitCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(processGroupNonZeroRateLimitCmd)

	processGroupCancelWithChildrenCmd := &cobra.Command{
		Use:   "process-group-cancel-children",
		Short: "Test cancellation terminates whole process group (no surviving children)",
		Args:  cobra.ExactArgs(0),
		Run:   cmdProcessGroupCancelWithChildren,
	}
	processGroupCancelWithChildrenCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(processGroupCancelWithChildrenCmd)

	processGroupMalformedBeforeFinalCmd := &cobra.Command{
		Use:   "process-group-malformed-before",
		Short: "Test malformed output before final event should still fail",
		Args:  cobra.ExactArgs(0),
		Run:   cmdProcessGroupMalformedBeforeFinal,
	}
	processGroupMalformedBeforeFinalCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(processGroupMalformedBeforeFinalCmd)

	processGroupMalformedAfterFinalCmd := &cobra.Command{
		Use:   "process-group-malformed-after",
		Short: "Test final event followed by malformed output should still succeed",
		Args:  cobra.ExactArgs(0),
		Run:   cmdProcessGroupMalformedAfterFinal,
	}
	processGroupMalformedAfterFinalCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(processGroupMalformedAfterFinalCmd)

	processGroupFinalDelayedCmd := &cobra.Command{
		Use:   "process-group-final-delayed",
		Short: "Test final event arrives before process termination (event wins)",
		Args:  cobra.ExactArgs(0),
		Run:   cmdProcessGroupFinalDelayed,
	}
	processGroupFinalDelayedCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(processGroupFinalDelayedCmd)

	healthModelCmd := &cobra.Command{
		Use:   "health-model",
		Short: "Test model health check",
		Args:  cobra.ExactArgs(0),
		Run:   cmdHealthModel,
	}
	healthModelCmd.Flags().String("log-dir", "", "Log output directory")
	healthModelCmd.Flags().String("model", "", "Model to check")
	root.AddCommand(healthModelCmd)

	smokeAllCmd := &cobra.Command{
		Use:   "smoke-all",
		Short: "Run all stable real scenarios sequentially",
		Args:  cobra.ExactArgs(0),
		Run:   cmdSmokeAll,
	}
	smokeAllCmd.Flags().String("model", "opencode/deepseek-v4-flash-free", "Model to use")
	smokeAllCmd.Flags().String("log-dir", "", "Log output directory")
	smokeAllCmd.Flags().Bool("verify-evidence", false, "Run verify-evidence after smoke-all completes")
	root.AddCommand(smokeAllCmd)

	verifyEvidenceCmd := &cobra.Command{
		Use:   "verify-evidence",
		Short: "Verify evidence completeness for a smoke run directory",
		Args:  cobra.ExactArgs(1),
		Run:   cmdVerifyEvidence,
	}
	verifyEvidenceCmd.Flags().Bool("strict", false, "Exit with failure code if any evidence issues found")
	root.AddCommand(verifyEvidenceCmd)

	dbUnavailableCmd := &cobra.Command{Use: "db-unavailable", Short: "Fake OpenCode DB unavailable smoke", Args: cobra.ExactArgs(0), Run: cmdDBUnavailable}
	dbUnavailableCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(dbUnavailableCmd)

	dbNonJSONCmd := &cobra.Command{Use: "db-non-json", Short: "Fake OpenCode DB non-JSON smoke", Args: cobra.ExactArgs(0), Run: cmdDBNonJSON}
	dbNonJSONCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(dbNonJSONCmd)

	dbTimeoutCmd := &cobra.Command{Use: "db-timeout", Short: "Fake OpenCode DB timeout smoke", Args: cobra.ExactArgs(0), Run: cmdDBTimeout}
	dbTimeoutCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(dbTimeoutCmd)

	dbLockedCmd := &cobra.Command{Use: "db-locked", Short: "Fake OpenCode DB locked smoke", Args: cobra.ExactArgs(0), Run: cmdDBLocked}
	dbLockedCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(dbLockedCmd)

	dbSessionNoAssistantCmd := &cobra.Command{Use: "db-session-no-assistant", Short: "Fake DB session without assistant smoke", Args: cobra.ExactArgs(0), Run: cmdDBSessionNoAssistant}
	dbSessionNoAssistantCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(dbSessionNoAssistantCmd)

	dbAssistantNoFinishCmd := &cobra.Command{Use: "db-assistant-no-finish", Short: "Fake DB assistant without finish smoke", Args: cobra.ExactArgs(0), Run: cmdDBAssistantNoFinish}
	dbAssistantNoFinishCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(dbAssistantNoFinishCmd)

	dbAssistantWithFinishCmd := &cobra.Command{Use: "db-assistant-with-finish", Short: "Fake DB assistant with finish smoke", Args: cobra.ExactArgs(0), Run: cmdDBAssistantWithFinish}
	dbAssistantWithFinishCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(dbAssistantWithFinishCmd)

	dbOnlyProofCmd := &cobra.Command{Use: "db-only-proof", Short: "Fake DB-only completion proof smoke", Args: cobra.ExactArgs(0), Run: cmdDBOnlyProof}
	dbOnlyProofCmd.Flags().String("log-dir", "", "Log output directory")
	root.AddCommand(dbOnlyProofCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type studyInfo struct {
	Name       string
	Path       string
	Sources    []string
	Dimensions []struct {
		Number string
		Name   string
		Title  string
		File   string
	}
}

func loadStudyInfo(studyPath string) (*studyInfo, error) {
	info := &studyInfo{Path: studyPath}

	if _, err := os.ReadDir(studyPath); err != nil {
		return nil, err
	}
	info.Name = filepath.Base(studyPath)

	sourcesPath := filepath.Join(studyPath, "sources")
	if srcEntries, err := os.ReadDir(sourcesPath); err == nil {
		for _, e := range srcEntries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				info.Sources = append(info.Sources, e.Name())
			}
		}
	}

	dimPath := filepath.Join(studyPath, "dimensions")
	if dimEntries, err := os.ReadDir(dimPath); err == nil {
		for _, e := range dimEntries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				content, _ := os.ReadFile(filepath.Join(dimPath, e.Name()))
				title := parseDimensionTitle(string(content))
				parts := strings.SplitN(strings.TrimSuffix(e.Name(), ".md"), "-", 2)
				info.Dimensions = append(info.Dimensions, struct {
					Number string
					Name   string
					Title  string
					File   string
				}{
					Number: parts[0],
					Name:   parts[1],
					Title:  title,
					File:   e.Name(),
				})
			}
		}
	}

	return info, nil
}

func parseDimensionTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# Dimension:") {
			return strings.TrimPrefix(line, "# Dimension:")
		}
	}
	return ""
}

func cmdList(cmd *cobra.Command, args []string) {
	studyName := args[0]
	studyPath := filepath.Join(UltraPlanRoot, "studies", studyName)

	info, err := loadStudyInfo(studyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading study: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Study: %s ===\n\n", info.Name)

	fmt.Println("Sources:")
	for _, s := range info.Sources {
		fmt.Printf("  - %s\n", s)
	}

	fmt.Println("\nDimensions:")
	for _, d := range info.Dimensions {
		fmt.Printf("  %s-%s: %s\n", d.Number, d.Name, d.Title)
	}
}

func cmdRun(cmd *cobra.Command, args []string) {
	cfg := runConfig{
		studyName:   args[0],
		dimension:   args[1],
		source:      args[2],
		maxAttempts: 3,
		timeoutMs:   1800000,
		validate:    true,
		observe:     true,
		healthCheck: true,
	}

	if model, _ := cmd.Flags().GetString("model"); model != "" {
		cfg.model = model
	}
	if variant, _ := cmd.Flags().GetString("variant"); variant != "" {
		cfg.variant = variant
	}
	if session, _ := cmd.Flags().GetString("session"); session != "" {
		cfg.sessionID = session
	}
	if sessionCont, _ := cmd.Flags().GetBool("session-continue"); sessionCont {
		cfg.sessionCont = true
	}
	if maxAttempts, _ := cmd.Flags().GetInt("max-attempts"); maxAttempts > 0 {
		cfg.maxAttempts = maxAttempts
	}
	if validate, _ := cmd.Flags().GetBool("validate"); !validate {
		cfg.validate = false
	}
	if observe, _ := cmd.Flags().GetBool("observe"); !observe {
		cfg.observe = false
	}
	if healthCheck, _ := cmd.Flags().GetBool("health-check"); !healthCheck {
		cfg.healthCheck = false
	}
	if logDir, _ := cmd.Flags().GetString("log-dir"); logDir != "" {
		cfg.logDir = logDir
	}

	studyPath := filepath.Join(UltraPlanRoot, "studies", cfg.studyName)
	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))

	if cfg.model == "" {
		cfg.model = cfgJSON.SprintExecutionModel
	}
	if cfg.variant == "" {
		cfg.variant = cfgJSON.DefaultVariant
	}

	if cfg.logDir == "" {
		cfg.logDir = filepath.Join(studyPath, ".agentwrap-logs", time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(cfg.logDir, 0755)

	logFile := filepath.Join(cfg.logDir, "run.log")
	f, err := os.Create(logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	logger := &logWriter{w: f}
	runStudyWithAgentwrap(cmd.Context(), &cfg, studyPath, logger)
}

type logWriter struct {
	w io.Writer
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return lw.w.Write(p)
}

func runStudyWithAgentwrap(ctx context.Context, cfg *runConfig, studyPath string, logger io.Writer) {
	startTime := time.Now()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		logger.Write([]byte(msg + "\n"))
	}

	sessionLabel := cfg.sessionID
	if sessionLabel == "" {
		sessionLabel = "new"
	}

	log("=== AGENTWRAP STUDY RUN ===")
	log("Study: %s | Dimension: %s | Source: %s", cfg.studyName, cfg.dimension, cfg.source)
	log("Model: %s | Variant: %s | Session: %s", cfg.model, cfg.variant, sessionLabel)
	log("Features: validate=%v observe=%v health=%v", cfg.validate, cfg.observe, cfg.healthCheck)
	log("Log dir: %s", cfg.logDir)
	log("Started: %s", startTime.Format(time.RFC3339))
	log("")

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	baseRuntime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	var runtime agentwrap.Runtime = baseRuntime

	if cfg.healthCheck {
		log("--- HEALTH CHECKS ---")
		healthReq := agentwrap.HealthCheckRequest{
			Context: agentwrap.RuntimeContext{
				RuntimeKind: agentwrap.RuntimeKind("opencode"),
				Provider:    agentwrap.ProviderID("opencode"),
				Model:       agentwrap.ModelID(cfg.model),
			},
			WorkDir: studyPath,
			Checks: []agentwrap.HealthCheckID{
				agentwrap.HealthCheckRuntimeAvailable,
				agentwrap.HealthCheckStructuredOutput,
				agentwrap.HealthCheckWorkDir,
				agentwrap.HealthCheckConfig,
				agentwrap.HealthCheckProvider,
				agentwrap.HealthCheckModel,
			},
			RequiredChecks: []agentwrap.HealthCheckID{
				agentwrap.HealthCheckRuntimeAvailable,
			},
		}

		report, err := baseRuntime.CheckHealth(ctx, healthReq)
		if err != nil {
			log("Health check error: %v", err)
		} else {
			log("Health status: %s", report.OverallStatus)
			log("  %d health checks performed", len(report.Results))
			for _, check := range report.Results {
				log("  %s: %s", check.Check, check.Status)
				if check.UserDetail != "" {
					log("    -> %s", check.UserDetail)
				}
			}

			if failure := agentwrap.RequiredHealthFailure(report, healthReq.RequiredChecks); failure != nil {
				log("REQUIRED HEALTH CHECK FAILED: %s", failure.Error())
				log("Proceeding anyway for testing purposes...")
			}
		}
		log("")
	}

	var memoryStore *agentwrap.MemoryRunStore
	var eventSink *capturingSink

	if cfg.observe {
		log("--- OBSERVABILITY SETUP ---")
		memoryStore = agentwrap.NewMemoryRunStore()
		eventSink = &capturingSink{events: make([]agentwrap.RunEventRecord, 0)}

		runtime = agentwrap.ObservingRuntime{
			Runtime: runtime,
			Store:   memoryStore,
			Sinks: []agentwrap.NamedEventSink{
				{
					Name:     "capturing",
					Sink:     eventSink,
					Required: false,
				},
			},
		}
		log("ObservingRuntime configured with MemoryRunStore and capturing sink")
		log("")
	}

	policy := agentwrap.BasicPolicy{
		MaxAttemptsPerTarget: cfg.maxAttempts,
		MaxElapsed:           30 * time.Minute,
		Backoff: agentwrap.ExponentialBackoff{
			Initial: 10 * time.Second,
			Factor:  2,
			Max:     5 * time.Minute,
		},
		RetryRateLimits: true,
	}

	fallbackRuntime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	policy.Fallbacks = []agentwrap.FallbackAlternative{
		{
			Name:    "backup-model",
			Runtime: fallbackRuntime,
			Request: agentwrap.RunRequest{
				Provider: agentwrap.ProviderID("opencode"),
				Model:    agentwrap.ModelID("opencode/deepseek-v4-flash-free"),
			},
		},
	}

	runner := agentwrap.PolicyRunner{
		Runtime:      runtime,
		Policy:       policy,
		Alternatives: policy.Fallbacks,
	}

	log("--- POLICY RUNNER ---")
	log("Max attempts: %d", policy.MaxAttemptsPerTarget)
	if expB, ok := policy.Backoff.(agentwrap.ExponentialBackoff); ok {
		log("Backoff: Initial=%v Factor=%v Max=%v", expB.Initial, expB.Factor, expB.Max)
	}
	log("Retry rate limits: %v", policy.RetryRateLimits)
	log("Fallback alternatives: %d", len(policy.Fallbacks))
	log("")

	if cfg.validate {
		log("--- VALIDATING RUNTIME ---")
		validating := agentwrap.ValidatingRuntime{
			Runtime: runner,
			Spec: agentwrap.ValidationSpec{
				Expectations: []agentwrap.ValidationExpectation{
					{
						ID:       "report-exists",
						Kind:     agentwrap.ExpectationFile,
						Path:     fmt.Sprintf("reports/source/%s/%s.md", cfg.dimension, cfg.source),
						Severity: agentwrap.ExpectationRequired,
					},
				},
				Repair: agentwrap.RepairConfig{
					MaxAttempts:   2,
					SessionAction: agentwrap.SessionActionContinue,
					ShouldRepair: func(ctx agentwrap.RepairContext) bool {
						return true
					},
					BuildPrompt: func(ctx agentwrap.RepairContext) string {
						return "Please fix the validation failures and rewrite the output files."
					},
				},
			},
		}
		log("Validation enabled with %d expectations", len(validating.Spec.Expectations))
		log("Repair: MaxAttempts=%d SessionAction=%s",
			validating.Spec.Repair.MaxAttempts, validating.Spec.Repair.SessionAction)
		log("")
		runtime = validating
	} else {
		runtime = runner
	}

	prompt := buildStudyPrompt(cfg, studyPath)
	promptFile := filepath.Join(cfg.logDir, "prompt.txt")
	os.WriteFile(promptFile, []byte(prompt), 0644)
	log("Prompt written to: %s", promptFile)
	log("")

	runReq := agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  studyPath,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(cfg.model),
		Timeout:  time.Duration(cfg.timeoutMs) * time.Millisecond,
		PermissionPolicy: &agentwrap.PermissionPolicy{
			Default: agentwrap.PermissionActionAsk,
			Tools: map[agentwrap.PermissionTool]agentwrap.PermissionAction{
				agentwrap.PermissionToolRead:   agentwrap.PermissionActionAllow,
				agentwrap.PermissionToolEdit:   agentwrap.PermissionActionAllow,
				agentwrap.PermissionToolShell:  agentwrap.PermissionActionAsk,
				agentwrap.PermissionToolGlob:   agentwrap.PermissionActionAllow,
				agentwrap.PermissionToolSearch: agentwrap.PermissionActionAllow,
			},
			UnsupportedBehavior: agentwrap.PermissionUnsupportedBestEffort,
		},
		RequireHealth: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckRuntimeAvailable,
		},
	}

	if cfg.sessionID != "" && cfg.sessionCont {
		runReq.WantSession = true
		runReq.SessionAction = agentwrap.SessionActionContinue
		runReq.SessionID = agentwrap.SessionID(cfg.sessionID)
	}

	log("--- STARTING RUN ---")
	log("Provider: %s | Model: %s | Timeout: %v",
		runReq.Provider, runReq.Model, runReq.Timeout)
	if runReq.WantSession {
		log("Session: ID=%s Action=%s", runReq.SessionID, runReq.SessionAction)
	}
	log("")

	run, err := runtime.StartRun(ctx, runReq)
	if err != nil {
		log("ERROR starting run: %v", err)
		saveResults(cfg, startTime, agentwrap.RunResult{}, err)
		os.Exit(1)
	}

	runID := run.ID()
	log("Run started: ID=%s", runID)
	log("")

	eventCount := 0
	eventLogFile := filepath.Join(cfg.logDir, "events.jsonl")
	ef, _ := os.Create(eventLogFile)
	defer ef.Close()

	eventCh := run.Events()
	if eventCh != nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-eventCh:
					if !ok {
						return
					}
					eventCount++
					if ev.Type != "" {
						ef.WriteString(fmt.Sprintf("%s\n", jsonMustMarshal(ev)))
					}
				}
			}
		}()
	}

	result, err := run.Wait(ctx)
	waitTime := time.Since(startTime)

	log("")
	log("--- RUN COMPLETE ---")
	log("Run ID: %s", runID)
	log("Wait time: %v", waitTime)

	if err != nil {
		log("Error: %v", err)
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("  Category: %s", sdkErr.Category)
			log("  UserDetail: %s", sdkErr.UserDetail)
		}
	} else {
		log("Status: %s", result.Status)
		if result.SessionID != "" {
			log("Session ID: %s", result.SessionID)
		}
		if result.Usage.InputTokens != nil {
			log("Usage: Input=%d Output=%d Total=%d",
				*result.Usage.InputTokens, *result.Usage.OutputTokens, *result.Usage.TotalTokens)
		}
	}

	if cfg.observe && memoryStore != nil {
		log("")
		log("--- OBSERVABILITY DATA ---")
		ctx := context.Background()
		if completed, found, err := memoryStore.GetCompletedRun(ctx, runID); err == nil && found {
			log("Stored run status: %s", completed.Status)
			log("Stored run duration: %v", completed.Duration)
		}

		if events, err := memoryStore.ListRunEvents(ctx, runID); err == nil {
			log("Stored events for run: %d", len(events))
		}

		if activeRuns, err := memoryStore.ListActiveRuns(ctx); err == nil {
			log("Active runs in store: %d", len(activeRuns))
		}

		if eventSink != nil {
			log("Captured events: %d", len(eventSink.events))
			for i, ev := range eventSink.events {
				if i < 5 || i >= len(eventSink.events)-3 {
					log("  Event %d: type=%s kind=%s", i, ev.Type, ev.Kind)
				} else if i == 5 {
					log("  ... (%d more events)", len(eventSink.events)-8)
				}
			}
		}
	}

	saveResults(cfg, startTime, result, err)

	log("")
	log("=== RUN COMPLETE ===")
	log("Total time: %v", time.Since(startTime))
	log("Log files: %s", cfg.logDir)
}

func buildStudyPrompt(cfg *runConfig, studyPath string) string {
	var sb strings.Builder

	sb.WriteString("# Study: Project Structure Analysis\n\n")
	sb.WriteString("Study the source code following the dimension instructions.\n\n")

	dimFile := filepath.Join(studyPath, "dimensions", cfg.dimension+".md")
	if content, err := os.ReadFile(dimFile); err == nil {
		sb.WriteString("## Dimension\n\n")
		sb.WriteString(string(content))
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Source\n\n")
	sb.WriteString(fmt.Sprintf("Study source: %s\n\n", cfg.source))

	sourcePath := filepath.Join(studyPath, "sources", cfg.source)
	sb.WriteString(fmt.Sprintf("Source path: %s\n\n", sourcePath))

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("1. Explore the source code following the dimension guidance.\n")
	sb.WriteString("2. Answer all questions in the dimension.\n")
	sb.WriteString(fmt.Sprintf("3. Write the analysis to `reports/source/%s/%s.md`.\n", cfg.dimension, cfg.source))
	sb.WriteString("\n")

	templateFile := filepath.Join(UltraPlanRoot, "templates", "repo-analysis.md")
	if content, err := os.ReadFile(templateFile); err == nil {
		sb.WriteString("## Output Template\n\n")
		sb.WriteString(string(content))
	}

	return sb.String()
}

type capturingSink struct {
	events []agentwrap.RunEventRecord
}

func (cs *capturingSink) AppendEvent(ctx context.Context, event agentwrap.RunEventRecord) error {
	cs.events = append(cs.events, event)
	return nil
}

func jsonMustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func saveResults(cfg *runConfig, startTime time.Time, result agentwrap.RunResult, runErr error) {
	resultsFile := filepath.Join(cfg.logDir, "results.json")

	type results struct {
		Config       runConfig                     `json:"config"`
		StartTime    time.Time                     `json:"start_time"`
		EndTime      time.Time                     `json:"end_time"`
		Duration     time.Duration                 `json:"duration"`
		RunID        string                        `json:"run_id,omitempty"`
		SessionID    string                        `json:"session_id,omitempty"`
		Error        string                        `json:"error,omitempty"`
		Status       string                        `json:"status,omitempty"`
		Warnings     []string                      `json:"warnings,omitempty"`
		Usage        *agentwrap.Usage              `json:"usage,omitempty"`
		RetryCount   int                           `json:"retry_count,omitempty"`
		FallbackUsed bool                          `json:"fallback_used,omitempty"`
		Attempts     []agentwrap.AttemptSummary    `json:"attempts,omitempty"`
		Metadata     *agentwrap.RunMetadata        `json:"metadata,omitempty"`
		Artifacts    []agentwrap.ArtifactRef       `json:"artifacts,omitempty"`
		NativeMeta   map[string]any                `json:"native_metadata,omitempty"`
		Policy       *agentwrap.PolicyMetadata     `json:"policy_metadata,omitempty"`
		Validation   *agentwrap.ValidationMetadata `json:"validation_metadata,omitempty"`
		Repair       *agentwrap.RepairMetadata     `json:"repair_metadata,omitempty"`
		Cleanup      *agentwrap.CleanupMetadata    `json:"cleanup_metadata,omitempty"`
		Category     string                        `json:"error_category,omitempty"`
		EventLog     string                        `json:"event_log,omitempty"`
		EventLogOK   bool                          `json:"event_log_present,omitempty"`
	}

	r := results{
		Config:    *cfg,
		StartTime: startTime,
		EndTime:   time.Now(),
		Duration:  time.Since(startTime),
	}

	if runErr != nil {
		r.Error = runErr.Error()
		var sdkErr *agentwrap.SDKError
		if errors.As(runErr, &sdkErr) {
			r.Category = string(sdkErr.Category)
		}
	}
	if result.Status != "" {
		r.Status = string(result.Status)
		r.RunID = string(result.RunID)
		r.SessionID = string(result.SessionID)
		r.Warnings = result.Warnings
		r.Artifacts = result.Artifacts
		r.Attempts = result.Metadata.Attempts
		if len(result.Metadata.Attempts) > 1 {
			r.FallbackUsed = true
			r.RetryCount = len(result.Metadata.Attempts) - 1
		}
		if result.Usage.InputTokens != nil || result.Usage.OutputTokens != nil || result.Usage.TotalTokens != nil {
			u := result.Usage
			r.Usage = &u
		}
		r.Metadata = &result.Metadata
		r.NativeMeta = result.Metadata.NativeMetadata
		r.Policy = &result.Metadata.Policy
		if result.Metadata.Validation.Configured {
			r.Validation = &result.Metadata.Validation
		}
		if result.Metadata.Repair.Configured {
			r.Repair = &result.Metadata.Repair
		}
		r.Cleanup = &result.Metadata.Cleanup
	}
	eventLog := filepath.Join(cfg.logDir, "events.jsonl")
	if _, err := os.Stat(eventLog); err == nil {
		r.EventLog = eventLog
		r.EventLogOK = true
	}

	os.WriteFile(resultsFile, []byte(jsonMustMarshal(r)), 0644)
}

func checkExpectation(cfg *runConfig, result agentwrap.RunResult, err error) *scenarioResult {
	status := ""
	category := ""
	if result.Status != "" {
		status = string(result.Status)
	}
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			category = string(sdkErr.Category)
		} else {
			category = "unknown"
		}
		if status == "" {
			status = "failed"
		}
	}
	sr := &scenarioResult{
		Name:      "",
		Passed:    true,
		Status:    status,
		Category:  category,
		SessionID: string(result.SessionID),
		LogDir:    cfg.logDir,
		Warnings:  result.Warnings,
	}
	if result.Usage.InputTokens != nil || result.Usage.OutputTokens != nil || result.Usage.TotalTokens != nil {
		sr.Usage = &result.Usage
	}
	if cfg.expectStatus != "" {
		sr.Expect = &scenarioExpectation{Status: cfg.expectStatus, Category: cfg.expectCategory}
		if status != cfg.expectStatus {
			sr.Passed = false
			return sr
		}
	}
	if cfg.expectCategory != "" && category != cfg.expectCategory {
		sr.Passed = false
		return sr
	}
	return sr
}

func saveOpenCodeDB(cfg *runConfig, sessionID string) {
	statusFile := filepath.Join(cfg.logDir, "opencode-db-snapshot-status.json")
	type snapshotStatus struct {
		SessionID string            `json:"session_id,omitempty"`
		Status    string            `json:"status"`
		Files     map[string]string `json:"files,omitempty"`
		Errors    map[string]string `json:"errors,omitempty"`
	}
	status := snapshotStatus{
		SessionID: sessionID,
		Status:    "captured",
		Files:     map[string]string{},
		Errors:    map[string]string{},
	}
	if sessionID == "" {
		status.Status = "skipped"
		status.Errors["session"] = "no session id"
		os.WriteFile(statusFile, []byte(jsonMustMarshal(status)), 0644)
		return
	}
	queries := map[string]string{
		"opencode-session.json":  fmt.Sprintf("select * from session where id=%s", sqlString(sessionID)),
		"opencode-messages.json": fmt.Sprintf("select * from message where session_id=%s order by time_created", sqlString(sessionID)),
		"opencode-parts.json":    fmt.Sprintf("select * from part where session_id=%s order by time_created", sqlString(sessionID)),
	}
	dbTimeout := 5 * time.Second
	for filename, query := range queries {
		executable := cfg.dbExecutable
		if executable == "" {
			executable = "opencode"
		}
		ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
		cmd := exec.CommandContext(ctx, executable, "db", "--format", "json", query)
		if len(cfg.dbEnv) > 0 {
			cmd.Env = append(os.Environ(), cfg.dbEnv...)
		}
		out, err := cmd.Output()
		cancel()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				status.Status = "failed"
				status.Errors[filename] = "query timeout after " + dbTimeout.String()
				continue
			}
			status.Status = "failed"
			status.Errors[filename] = err.Error()
			continue
		}
		path := filepath.Join(cfg.logDir, filename)
		if err := os.WriteFile(path, out, 0644); err != nil {
			status.Status = "failed"
			status.Errors[filename] = "write failed: " + err.Error()
			continue
		}
		if !json.Valid(out) {
			status.Status = "failed"
			status.Errors[filename] = "not valid JSON"
			continue
		}
		status.Files[filename] = path
	}
	if status.Status == "captured" && len(status.Files) != len(queries) {
		status.Status = "partial"
	}
	os.WriteFile(statusFile, []byte(jsonMustMarshal(status)), 0644)
}

func sqlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func cmdSmokeAll(cmd *cobra.Command, args []string) {
	model, _ := cmd.Flags().GetString("model")
	logDir := filepath.Join(UltraPlanRoot, ".agentwrap-logs", "smoke-all-"+time.Now().Format("20060102-150405"))
	os.MkdirAll(logDir, 0755)

	results := []*scenarioResult{}
	allPassed := true

	type scenarioConfig struct {
		name     string
		cmdFn    func(*cobra.Command, []string)
		expect   scenarioExpectation
		setFlags func(*cobra.Command)
	}
	scenarios := []scenarioConfig{
		{name: "smoke-text", cmdFn: cmdSmokeText, expect: scenarioExpectation{Status: "completed"}},
		{name: "smoke-reasoning", cmdFn: cmdSmokeReasoning, expect: scenarioExpectation{Status: "completed"}},
		{name: "smoke-file-write", cmdFn: cmdSmokeFileWrite, expect: scenarioExpectation{Status: "completed"}},
		{name: "usage", cmdFn: cmdUsage, expect: scenarioExpectation{Status: "completed"}},
		{name: "artifacts", cmdFn: cmdArtifacts, expect: scenarioExpectation{Status: "completed"}},
		{name: "fixed-backoff", cmdFn: cmdFixedBackoff, expect: scenarioExpectation{Status: "completed"}},
		{name: "validate-json", cmdFn: cmdValidateJson, expect: scenarioExpectation{Status: "completed"}},
		{name: "validate-md", cmdFn: cmdValidateMd, expect: scenarioExpectation{Status: "completed"}},
		{name: "custom-validator", cmdFn: cmdCustomValidator, expect: scenarioExpectation{Status: "completed"}},
		{name: "validate-fail", cmdFn: cmdValidateFail, expect: scenarioExpectation{Status: "failed", Category: "validation"}},
		{name: "validate-repair", cmdFn: cmdValidateRepair, expect: scenarioExpectation{Status: "completed"}},
		{name: "validate-repair-exhaust", cmdFn: cmdValidateRepairExhaust, expect: scenarioExpectation{Status: "failed", Category: "repair_exhausted"}},
		{name: "cancel", cmdFn: cmdCancel, expect: scenarioExpectation{Status: "cancelled", Category: "cancellation"}},
		{name: "timeout", cmdFn: cmdTimeout, expect: scenarioExpectation{Status: "failed", Category: "timeout"},
			setFlags: func(c *cobra.Command) { c.Flags().Set("timeout-ms", "500") }},
		{name: "fallback-invalid-model", cmdFn: cmdFallbackInvalidModel, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"},
			setFlags: func(c *cobra.Command) {
				c.Flags().Set("primary-model", "opencode/not-a-real-model")
				c.Flags().Set("fallback-model", "opencode/deepseek-v4-flash-free")
			}},
		{name: "fallback-invalid-provider", cmdFn: cmdFallbackInvalidProvider, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"},
			setFlags: func(c *cobra.Command) {
				c.Flags().Set("primary-model", "not-a-provider/model")
				c.Flags().Set("fallback-model", "opencode/deepseek-v4-flash-free")
			}},
		{name: "fallback-all-fail", cmdFn: cmdFallbackAllFail, expect: scenarioExpectation{Status: "failed", Category: "model_unavailable"}},
		{name: "invalid-separate-provider", cmdFn: cmdInvalidSeparateProvider, expect: scenarioExpectation{Status: "failed", Category: "configuration"}},
		{name: "health-fail", cmdFn: cmdHealthFail, expect: scenarioExpectation{Status: "completed"}},
		{name: "health-model", cmdFn: cmdHealthModel, expect: scenarioExpectation{Status: "completed"}},

		{name: "invalid-provider", cmdFn: cmdInvalidProvider, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"},
			setFlags: func(c *cobra.Command) { c.Flags().Set("model", "nonexistent-provider/test") }},
		{name: "invalid-model", cmdFn: cmdInvalidModel, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"},
			setFlags: func(c *cobra.Command) { c.Flags().Set("model", "opencode/not-a-real-model") }},
		{name: "opencode-db-unavailable", cmdFn: cmdOpenCodeDBUnavailable, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"}},
		{name: "provider-429", cmdFn: cmdProvider429, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"},
			setFlags: func(c *cobra.Command) { c.Flags().Set("model", "minimax-coding-plan/MiniMax-M2.7") }},
		{name: "provider-auth", cmdFn: cmdProviderAuth, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"},
			setFlags: func(c *cobra.Command) { c.Flags().Set("model", "minimax-coding-plan/MiniMax-M2.7") }},
		{name: "fatal-clean", cmdFn: cmdFatalClean, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"}},
		{name: "session-fork", cmdFn: cmdSessionFork, expect: scenarioExpectation{Status: "failed", Category: "configuration"}},
		{name: "session-fresh", cmdFn: cmdSessionFresh, expect: scenarioExpectation{Status: "completed"}},
		{name: "session-continue-existing", cmdFn: cmdSessionContinueExisting, expect: scenarioExpectation{Status: "completed"}},
		{name: "session-continue-missing", cmdFn: cmdSessionContinueMissing, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"}},
		{name: "session-continue-after-fail", cmdFn: cmdSessionContinueAfterFail, expect: scenarioExpectation{Status: "completed"}},
		{name: "repair-with-continue", cmdFn: cmdRepairWithContinue, expect: scenarioExpectation{Status: "completed"}},
		{name: "db-unavailable", cmdFn: cmdDBUnavailable, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"}},
		{name: "db-non-json", cmdFn: cmdDBNonJSON, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"}},
		{name: "db-timeout", cmdFn: cmdDBTimeout, expect: scenarioExpectation{Status: "failed", Category: "runtime_exit"}},
		{name: "db-locked", cmdFn: cmdDBLocked, expect: scenarioExpectation{Status: "failed", Category: "runtime_unavailable"}},
		{name: "db-session-no-assistant", cmdFn: cmdDBSessionNoAssistant, expect: scenarioExpectation{Status: "completed"}},
		{name: "db-assistant-no-finish", cmdFn: cmdDBAssistantNoFinish, expect: scenarioExpectation{Status: "completed"}},
		{name: "db-assistant-with-finish", cmdFn: cmdDBAssistantWithFinish, expect: scenarioExpectation{Status: "completed"}},

		// Workstream 7: Process-Group Cleanup and Final-State Precedence
		{name: "process-group-nonzero-final", cmdFn: cmdProcessGroupNonZeroFinal, expect: scenarioExpectation{Status: "completed", Category: "completed"}},
		{name: "process-group-nonzero-ratelimit", cmdFn: cmdProcessGroupNonZeroRateLimit, expect: scenarioExpectation{Status: "failed", Category: "rate_limit"}},
		{name: "process-group-cancel-children", cmdFn: cmdProcessGroupCancelWithChildren, expect: scenarioExpectation{Status: "cancelled", Category: "cancellation"}},
		{name: "process-group-malformed-before", cmdFn: cmdProcessGroupMalformedBeforeFinal, expect: scenarioExpectation{Status: "failed", Category: "malformed_event"}},
		{name: "process-group-malformed-after", cmdFn: cmdProcessGroupMalformedAfterFinal, expect: scenarioExpectation{Status: "completed", Category: "completed"}},
		{name: "process-group-final-delayed", cmdFn: cmdProcessGroupFinalDelayed, expect: scenarioExpectation{Status: "completed", Category: "completed"}},
	}

	summaryF, _ := os.Create(filepath.Join(logDir, "summary.json"))
	defer summaryF.Close()

	for _, sc := range scenarios {
		scLogDir := filepath.Join(logDir, sc.name)
		os.MkdirAll(scLogDir, 0755)

		scArgs := []string{} // scenario-specific args, currently empty
		if model != "" && sc.name != "fallback-invalid-model" && sc.name != "fallback-invalid-provider" && sc.name != "fallback-all-fail" {
			scArgs = append(scArgs, "--model", model)
		}

		subCmd := &cobra.Command{Use: sc.name, Run: sc.cmdFn}
		subCmd.Flags().String("log-dir", "", "Log output directory")
		subCmd.Flags().String("model", "", "Model to use")
		subCmd.Flags().String("primary-model", "", "Primary model")
		subCmd.Flags().String("fallback-model", "", "Fallback model")
		subCmd.Flags().Int64("timeout-ms", 0, "Timeout in ms")
		subCmd.Flags().String("expect-status", "", "Expected final status")
		subCmd.Flags().String("expect-category", "", "Expected error category")

		for _, a := range scArgs {
			parts := strings.SplitN(a, "=", 2)
			if len(parts) == 2 {
				subCmd.Flags().Set(parts[0], parts[1])
			}
		}
		subCmd.Flags().Set("log-dir", scLogDir)
		if model != "" && sc.name != "fallback-invalid-model" && sc.name != "fallback-invalid-provider" && sc.name != "fallback-all-fail" {
			subCmd.Flags().Set("model", model)
		}
		if sc.setFlags != nil {
			sc.setFlags(subCmd)
		}

		subCmd.SetArgs(scArgs)
		sc.cmdFn(subCmd, scArgs)

		// Load saved result and check expectation
		data, err := os.ReadFile(filepath.Join(scLogDir, "results.json"))
		sr := &scenarioResult{Name: sc.name, LogDir: scLogDir, Passed: true, Expect: &sc.expect}
		if err == nil {
			var saved struct {
				Status    string `json:"status"`
				Category  string `json:"error_category,omitempty"`
				SessionID string `json:"session_id"`
				Error     string `json:"error,omitempty"`
			}
			json.Unmarshal(data, &saved)
			sr.Status = saved.Status
			sr.Category = saved.Category
			sr.SessionID = saved.SessionID
			sr.Err = saved.Error
		}
		// For commands that don't create runs (health-fail, session-fork), derive status from log
		if sr.Status == "" {
			if dirEntries, dirErr := os.ReadDir(scLogDir); dirErr == nil {
				for _, de := range dirEntries {
					if !de.IsDir() && strings.HasSuffix(de.Name(), ".log") {
						if logBytes, logErr := os.ReadFile(filepath.Join(scLogDir, de.Name())); logErr == nil {
							logText := string(logBytes)
							switch {
							case strings.Contains(logText, "Overall status:"):
								sr.Status = "completed"
							case strings.Contains(logText, "does not support session fork"):
								sr.Status = "failed"
								sr.Category = "configuration"
							case strings.Contains(logText, "unsupported"):
								sr.Status = "failed"
								if sr.Category == "" {
									sr.Category = "configuration"
								}
							}
						}
						break
					}
				}
			}
		}

		if sc.expect.Status != "" && sr.Status != sc.expect.Status {
			sr.Passed = false
		}
		if sc.expect.Category != "" && sr.Category != sc.expect.Category {
			sr.Passed = false
		}

		if !sr.Passed {
			allPassed = false
		}
		results = append(results, sr)

		entry, _ := json.Marshal(sr)
		summaryF.WriteString(string(entry) + "\n")
	}

	fmt.Println("\n=== SMOKE ALL SUMMARY ===")
	fmt.Printf("%-20s %-12s %-8s %s\n", "SCENARIO", "STATUS", "PASSED", "LOG DIR")
	fmt.Println(strings.Repeat("-", 80))
	for _, sr := range results {
		passStr := "PASS"
		if !sr.Passed {
			passStr = "FAIL"
		}
		fmt.Printf("%-20s %-12s %-8s %s\n", sr.Name, sr.Status, passStr, sr.LogDir)
	}
	fmt.Println(strings.Repeat("-", 80))
	if allPassed {
		fmt.Println("RESULT: ALL PASSED")
	} else {
		fmt.Println("RESULT: SOME SCENARIOS FAILED")
		os.Exit(1)
	}
}

func cmdSessionContinue(cmd *cobra.Command, args []string) {
	sessionID := args[0]
	newPrompt := args[1]

	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, "studies", ".agentwrap-logs", "continue-"+sessionID)
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "continue.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== SESSION CONTINUE ===")
	log("Session: %s", sessionID)
	log("New prompt: %s", newPrompt)
	log("Log dir: %s", logDir)
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	runtime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:        newPrompt,
		WorkDir:       UltraPlanRoot,
		Provider:      agentwrap.ProviderID("opencode"),
		Model:         agentwrap.ModelID(cfgJSON.SprintExecutionModel),
		Timeout:       30 * time.Minute,
		WantSession:   true,
		SessionAction: agentwrap.SessionActionContinue,
		SessionID:     agentwrap.SessionID(sessionID),
	})
	if err != nil {
		log("ERROR: %v", err)
		os.Exit(1)
	}

	log("Session continue run started: ID=%s", run.ID())

	result, err := run.Wait(context.Background())
	if err != nil {
		log("Error: %v", err)
	} else {
		log("Status: %s", result.Status)
		log("Session ID: %s", result.SessionID)
	}

	saveResults(&runConfig{
		sessionID: sessionID,
		logDir:    logDir,
	}, time.Now(), result, err)
}

func cmdCancel(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "cancel-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "cancel.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	modelStr, _ := cmd.Flags().GetString("model")
	expectStatus, _ := cmd.Flags().GetString("expect-status")
	expectCategory, _ := cmd.Flags().GetString("expect-category")

	rc := &runConfig{
		logDir:         logDir,
		expectStatus:   expectStatus,
		expectCategory: expectCategory,
	}

	log("=== CANCELLATION TEST ===")
	log("Log dir: %s", logDir)
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}
	log("Using model: %s", model)

	runtime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	prompt := "Count to 100 slowly. Say each number with a 2 second delay between each. Start now: 1, 2, 3..."

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR starting run: %v", err)
		os.Exit(1)
	}

	log("Run started: ID=%s", run.ID())
	log("Waiting 3 seconds before cancelling...")
	time.Sleep(3 * time.Second)

	log("Calling Run.Cancel()...")
	cancelErr := run.Cancel(context.Background())
	if cancelErr != nil {
		log("Cancel error: %v", cancelErr)
	} else {
		log("Cancel succeeded without error")
	}

	result, err := run.Wait(context.Background())
	if err != nil {
		log("Wait error: %v", err)
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("  Category: %s", sdkErr.Category)
			log("  UserDetail: %s", sdkErr.UserDetail)
		}
	}
	if result.Status != "" {
		log("Final Status: %s", result.Status)
	}
	for _, attempt := range result.Metadata.Attempts {
		log("Attempt %d target=%d model=%s status=%s error=%s",
			attempt.Attempt,
			attempt.TargetIndex,
			attempt.Request.Model,
			attempt.Status,
			attempt.ErrorCategory,
		)
		if attempt.RateLimit != nil {
			log("  RateLimit: provider=%s model=%s detail=%s",
				attempt.RateLimit.Provider,
				attempt.RateLimit.Model,
				attempt.RateLimit.UserDetail,
			)
		}
	}

	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
	sr := checkExpectation(rc, result, err)
	log("Expectation: status=%s category=%s passed=%v", rc.expectStatus, rc.expectCategory, sr.Passed)
}

func cmdTimeout(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	timeoutMs, _ := cmd.Flags().GetInt64("timeout-ms")
	expectStatus, _ := cmd.Flags().GetString("expect-status")
	expectCategory, _ := cmd.Flags().GetString("expect-category")

	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "timeout-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	rc := &runConfig{
		logDir:         logDir,
		expectStatus:   expectStatus,
		expectCategory: expectCategory,
	}

	logFile := filepath.Join(logDir, "timeout.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== TIMEOUT TEST ===")
	log("Log dir: %s", logDir)
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}
	log("Using model: %s", model)

	runtime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	prompt := "Count to 100 slowly. Say each number with a 2 second delay between each. Start now: 1, 2, 3..."

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  time.Duration(timeoutMs) * time.Millisecond,
	})
	if err != nil {
		log("ERROR starting run: %v", err)
		os.Exit(1)
	}

	log("Run started: ID=%s", run.ID())
	log("Timeout set to %dms", timeoutMs)

	result, err := run.Wait(context.Background())
	if err != nil {
		log("Wait error: %v", err)
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("  Category: %s", sdkErr.Category)
			log("  UserDetail: %s", sdkErr.UserDetail)
		}
	}
	if result.Status != "" {
		log("Final Status: %s", result.Status)
	}
	for _, attempt := range result.Metadata.Attempts {
		log("Attempt %d target=%d model=%s status=%s error=%s",
			attempt.Attempt,
			attempt.TargetIndex,
			attempt.Request.Model,
			attempt.Status,
			attempt.ErrorCategory,
		)
		if attempt.RateLimit != nil {
			log("  RateLimit: provider=%s model=%s detail=%s",
				attempt.RateLimit.Provider,
				attempt.RateLimit.Model,
				attempt.RateLimit.UserDetail,
			)
		}
	}

	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
	sr := checkExpectation(rc, result, err)
	log("Expectation: status=%s category=%s passed=%v", rc.expectStatus, rc.expectCategory, sr.Passed)
}

func cmdRateLimit(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	primaryModel, _ := cmd.Flags().GetString("primary-model")
	fallbackModel, _ := cmd.Flags().GetString("fallback-model")
	useFake, _ := cmd.Flags().GetBool("use-fake")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "ratelimit-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "ratelimit.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== RATE LIMIT FALLBACK TEST ===")
	log("Testing %s rate limit -> fallback to %s", primaryModel, fallbackModel)
	log("Log dir: %s", logDir)
	log("")

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	primary := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	fallback := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	runner := agentwrap.PolicyRunner{
		Runtime: primary,
		Policy: agentwrap.BasicPolicy{
			MaxAttemptsPerTarget: 1,
			RetryRateLimits:      true,
		},
		Alternatives: []agentwrap.FallbackAlternative{
			{
				Name:    "deepseek-fallback",
				Runtime: fallback,
				Request: agentwrap.RunRequest{
					Provider: agentwrap.ProviderID("opencode"),
					Model:    agentwrap.ModelID(fallbackModel),
				},
			},
		},
	}

	log("PolicyRunner configured with %s primary and %s fallback", primaryModel, fallbackModel)
	log("Starting run...")

	prompt := "Say hello in exactly 3 words."

	run, err := runner.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(primaryModel),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR starting run: %v", err)
		os.Exit(1)
	}

	log("Run started: ID=%s", run.ID())

	result, err := run.Wait(context.Background())
	if err != nil {
		log("Wait error: %v", err)
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("  Category: %s", sdkErr.Category)
			log("  UserDetail: %s", sdkErr.UserDetail)
		}
	}
	if result.Status != "" {
		log("Final Status: %s", result.Status)
	}

	for _, attempt := range result.Metadata.Attempts {
		log("Attempt %d target=%d model=%s status=%s error=%s",
			attempt.Attempt,
			attempt.TargetIndex,
			attempt.Request.Model,
			attempt.Status,
			attempt.ErrorCategory,
		)
		if attempt.RateLimit != nil {
			log("  RateLimit: provider=%s model=%s detail=%s",
				attempt.RateLimit.Provider,
				attempt.RateLimit.Model,
				attempt.RateLimit.UserDetail,
			)
		}
	}

	saveResults(&runConfig{
		logDir: logDir,
	}, time.Now(), result, err)
}

func cmdFixedBackoff(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "fixedbackoff-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "fixedbackoff.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== FIXED BACKOFF TEST ===")
	log("")

	modelStr, _ := cmd.Flags().GetString("model")
	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	primary := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}
	log("Using model: %s", model)

	runner := agentwrap.PolicyRunner{
		Runtime: primary,
		Policy: agentwrap.BasicPolicy{
			MaxAttemptsPerTarget: 2,
			RetryRateLimits:      false,
			Backoff: agentwrap.FixedBackoff{
				DelayValue: 2 * time.Second,
			},
		},
	}

	if bp, ok := runner.Policy.(agentwrap.BasicPolicy); ok {
		if fb, ok := bp.Backoff.(agentwrap.FixedBackoff); ok {
			log("FixedBackoff: DelayValue=%v", fb.DelayValue)
		}
	}

	prompt := "What is 2+2? Answer in one word."

	run, err := runner.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR starting run: %v", err)
		os.Exit(1)
	}

	log("Run started: ID=%s", run.ID())
	result, err := run.Wait(context.Background())
	if err != nil {
		log("Wait error: %v", err)
	}
	if result.Status != "" {
		log("Final Status: %s", result.Status)
	}

	saveResults(&runConfig{logDir: logDir}, time.Now(), result, err)
}

func cmdUsage(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "usage-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "usage.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== USAGE TRACKING TEST ===")
	log("")

	modelStr, _ := cmd.Flags().GetString("model")
	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	runtime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}
	log("Using model: %s", model)

	prompt := "What is 2+2? Answer in one word."

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR starting run: %v", err)
		os.Exit(1)
	}

	log("Run started: ID=%s", run.ID())
	result, err := run.Wait(context.Background())
	if err != nil {
		log("Wait error: %v", err)
	}
	if result.Usage.InputTokens != nil {
		log("Usage: Input=%d Output=%d Total=%d",
			*result.Usage.InputTokens, *result.Usage.OutputTokens, *result.Usage.TotalTokens)
	} else {
		log("Usage: no token counts returned")
	}
	if result.Status != "" {
		log("Final Status: %s", result.Status)
	}

	saveResults(&runConfig{logDir: logDir}, time.Now(), result, err)
}

func cmdArtifacts(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "artifacts-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "artifacts.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	eventFile := filepath.Join(logDir, "events.jsonl")
	ef, _ := os.Create(eventFile)
	defer ef.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== ARTIFACTS TEST ===")
	log("")

	modelStr, _ := cmd.Flags().GetString("model")
	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	runtime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}
	log("Using model: %s", model)

	prompt := "Create a simple markdown table with 2 columns: Name and Age. Put 2 rows of sample data."

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR starting run: %v", err)
		os.Exit(1)
	}

	log("Run started: ID=%s", run.ID())

	eventCh := run.Events()
	artifactCount := 0
	go func() {
		for ev := range eventCh {
			if ev.Type != "" {
				ef.WriteString(fmt.Sprintf("%s\n", jsonMustMarshal(ev)))
			}
			if ev.Kind() == agentwrap.EventArtifact || ev.Type == "artifact" {
				artifactCount++
			}
		}
	}()

	result, err := run.Wait(context.Background())
	if err != nil {
		log("Wait error: %v", err)
	}
	log("Artifacts observed during run: %d", artifactCount)
	if result.Status != "" {
		log("Final Status: %s", result.Status)
	}

	saveResults(&runConfig{logDir: logDir}, time.Now(), result, err)
}

func cmdValidateJson(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "validate-json-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	os.MkdirAll(filepath.Join(logDir, "reports"), 0755)

	logFile := filepath.Join(logDir, "validate-json.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== JSON VALIDATION TEST ===")
	log("")

	modelStr, _ := cmd.Flags().GetString("model")
	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	baseRuntime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	validating := agentwrap.ValidatingRuntime{
		Runtime: baseRuntime,
		Spec: agentwrap.ValidationSpec{
			Expectations: []agentwrap.ValidationExpectation{
				{
					ID:             "json-valid",
					Kind:           agentwrap.ExpectationJSON,
					Path:           "reports/test.json",
					Severity:       agentwrap.ExpectationRequired,
					RequiredFields: []string{"status", "score"},
				},
			},
		},
	}

	log("ValidatingRuntime with JSON expectation: reports/test.json must have status, score fields")
	log("Using model: %s", model)

	prompt := `Write a JSON file at reports/test.json with the following content:
{"status": "ok", "score": 100, "name": "test"}`

	run, err := validating.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  logDir,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR starting run: %v", err)
		os.Exit(1)
	}

	log("Run started: ID=%s", run.ID())
	result, err := run.Wait(context.Background())
	if err != nil {
		log("Wait error: %v", err)
	}
	if result.Status != "" {
		log("Final Status: %s", result.Status)
	}

	saveResults(&runConfig{logDir: logDir}, time.Now(), result, err)
}

func cmdValidateMd(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "validate-md-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	templatePath := filepath.Join(logDir, "template.md")
	os.WriteFile(templatePath, []byte("# Report\n\n## Summary\n\n[PLACEHOLDER: summarize here]\n\n## Details\n\n[PLACEHOLDER: add details here]\n"), 0644)

	logFile := filepath.Join(logDir, "validate-md.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== MARKDOWN TEMPLATE VALIDATION TEST ===")
	log("")

	modelStr, _ := cmd.Flags().GetString("model")
	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	baseRuntime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	validating := agentwrap.ValidatingRuntime{
		Runtime: baseRuntime,
		Spec: agentwrap.ValidationSpec{
			Expectations: []agentwrap.ValidationExpectation{
				{
					ID:           "md-template",
					Kind:         agentwrap.ExpectationMarkdownTemplate,
					Path:         "reports/test.md",
					TemplatePath: templatePath,
					Severity:     agentwrap.ExpectationRequired,
				},
			},
		},
	}

	log("ValidatingRuntime with MarkdownTemplate expectation")
	log("Using model: %s", model)

	templateContent, _ := os.ReadFile(templatePath)
	prompt := fmt.Sprintf(`Write a markdown file at reports/test.md following this template:
%s

Fill in the placeholders with actual content.`, string(templateContent))

	run, err := validating.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  logDir,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR starting run: %v", err)
		os.Exit(1)
	}

	log("Run started: ID=%s", run.ID())
	result, err := run.Wait(context.Background())
	if err != nil {
		log("Wait error: %v", err)
	}
	if result.Status != "" {
		log("Final Status: %s", result.Status)
	}

	saveResults(&runConfig{logDir: logDir}, time.Now(), result, err)
}

func cmdCustomValidator(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "custom-validator-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "custom-validator.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== CUSTOM VALIDATOR TEST ===")
	log("")

	modelStr, _ := cmd.Flags().GetString("model")
	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	baseRuntime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	customCheck := agentwrap.ValidatorFunc(func(ctx context.Context, vctx agentwrap.ValidationContext) agentwrap.ValidationCheck {
		content, err := os.ReadFile(filepath.Join(vctx.WorkDir, "reports", "custom.txt"))
		if err != nil {
			return agentwrap.ValidationCheck{
				ExpectationID: "custom-check",
				Detail:        "custom.txt not found",
				Passed:        false,
			}
		}
		if strings.TrimSpace(string(content)) == "VALID" {
			return agentwrap.ValidationCheck{
				ExpectationID: "custom-check",
				Detail:        "content matches expected",
				Passed:        true,
			}
		}
		return agentwrap.ValidationCheck{
			ExpectationID: "custom-check",
			Detail:        fmt.Sprintf("content mismatch: got %q", string(content)),
			Passed:        false,
		}
	})

	validating := agentwrap.ValidatingRuntime{
		Runtime: baseRuntime,
		Spec: agentwrap.ValidationSpec{
			Expectations: []agentwrap.ValidationExpectation{
				{
					ID:       "file-exists",
					Kind:     agentwrap.ExpectationFile,
					Path:     "reports/custom.txt",
					Severity: agentwrap.ExpectationRequired,
				},
			},
			Validators: []agentwrap.Validator{customCheck},
		},
	}

	log("ValidatingRuntime with custom ValidatorFunc")
	log("Using model: %s", model)

	prompt := `Write "VALID" (exactly, no quotes) to reports/custom.txt`

	run, err := validating.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   prompt,
		WorkDir:  logDir,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR starting run: %v", err)
		os.Exit(1)
	}

	log("Run started: ID=%s", run.ID())
	result, err := run.Wait(context.Background())
	if err != nil {
		log("Wait error: %v", err)
	}
	if result.Status != "" {
		log("Final Status: %s", result.Status)
	}

	saveResults(&runConfig{logDir: logDir}, time.Now(), result, err)
}

func cmdHealthFail(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "healthfail-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "healthfail.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== HEALTH CHECK FAILURE TEST ===")
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	runtime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	log("--- TEST 1: Nonexistent workdir ---")
	report1, err := runtime.CheckHealth(context.Background(), agentwrap.HealthCheckRequest{
		Context: agentwrap.RuntimeContext{
			RuntimeKind: agentwrap.RuntimeKind("opencode"),
			Provider:    agentwrap.ProviderID("opencode"),
			Model:       agentwrap.ModelID(cfgJSON.SprintExecutionModel),
		},
		WorkDir: "/nonexistent/path/that/does/not/exist",
		Checks: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckWorkDir,
		},
		RequiredChecks: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckWorkDir,
		},
	})
	if err != nil {
		log("Health check error: %v", err)
	}
	log("Overall status: %s", report1.OverallStatus)
	for _, r := range report1.Results {
		log("  %s: %s - %s", r.Check, r.Status, r.UserDetail)
	}

	log("")
	log("--- TEST 2: Bad config ---")
	badConfigPath := filepath.Join(logDir, "bad-config.json")
	os.WriteFile(badConfigPath, []byte(`{"invalid": json}`), 0644)

	originalConfig := os.Getenv("OPENCODE_CONFIG")
	os.Setenv("OPENCODE_CONFIG", badConfigPath)
	defer func() {
		if originalConfig != "" {
			os.Setenv("OPENCODE_CONFIG", originalConfig)
		}
	}()

	runtime2 := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + badConfigPath),
	)

	report2, err := runtime2.CheckHealth(context.Background(), agentwrap.HealthCheckRequest{
		Context: agentwrap.RuntimeContext{
			RuntimeKind: agentwrap.RuntimeKind("opencode"),
			Provider:    agentwrap.ProviderID("opencode"),
			Model:       agentwrap.ModelID(cfgJSON.SprintExecutionModel),
		},
		WorkDir: UltraPlanRoot,
		Checks: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckConfig,
		},
		RequiredChecks: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckConfig,
		},
	})
	if err != nil {
		log("Health check error: %v", err)
	}
	log("Overall status: %s", report2.OverallStatus)
	for _, r := range report2.Results {
		log("  %s: %s - %s", r.Check, r.Status, r.UserDetail)
	}

	saveResults(&runConfig{logDir: logDir}, time.Now(), agentwrap.RunResult{}, nil)
}

func cmdSessionFork(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "sessionfork-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "sessionfork.log")
	f, _ := os.Create(logFile)
	defer f.Close()

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== SESSION FORK TEST (unsupported) ===")
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	runtime := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath),
	)

	caps, err := runtime.Capabilities(context.Background())
	if err != nil {
		log("Capabilities error: %v", err)
	} else {
		log("Capabilities:")
		log("  session_fork supported: %v", caps.Supports(agentwrap.CapabilitySessionFork))
		log("  session_continue supported: %v", caps.Supports(agentwrap.CapabilitySessionContinue))
	}

	log("")
	log("Attempting SessionFork...")

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:        "Say hello in 3 words",
		WorkDir:       UltraPlanRoot,
		Provider:      agentwrap.ProviderID("opencode"),
		Model:         agentwrap.ModelID(cfgJSON.SprintExecutionModel),
		Timeout:       30 * time.Second,
		WantSession:   true,
		SessionAction: agentwrap.SessionActionFork,
	})
	if err != nil {
		log("StartRun error: %v", err)
	} else {
		result, err := run.Wait(context.Background())
		if err != nil {
			log("Wait error: %v", err)
		}
		log("Status: %s", result.Status)
		log("SessionID: %s", result.SessionID)
	}

	saveResults(&runConfig{logDir: logDir}, time.Now(), agentwrap.RunResult{}, err)
}

func cmdSmokeText(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "smoke-text-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "smoke-text.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Reply with exactly: OK",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log("ERROR: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if result.SessionID != "" {
		log("Session ID: %s", result.SessionID)
	}
	if result.Usage.InputTokens != nil {
		log("Usage: Input=%d Output=%d Total=%d", *result.Usage.InputTokens, *result.Usage.OutputTokens, *result.Usage.TotalTokens)
	}
	if err != nil {
		log("Error: %v", err)
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdSmokeReasoning(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "smoke-reasoning-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "smoke-reasoning.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Think briefly, then answer with exactly: OK",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if result.SessionID != "" {
		log("Session ID: %s", result.SessionID)
	}
	if result.Usage.InputTokens != nil {
		log("Usage: Input=%d Output=%d Total=%d", *result.Usage.InputTokens, *result.Usage.OutputTokens, *result.Usage.TotalTokens)
	}
	if err != nil {
		log("Error: %v", err)
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdSmokeFileWrite(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "smoke-file-write-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(filepath.Join(logDir, "reports"), 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "smoke-file-write.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Create reports/smoke.txt containing exactly: OK",
		WorkDir:  logDir,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("ERROR: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if result.SessionID != "" {
		log("Session ID: %s", result.SessionID)
	}
	if err != nil {
		log("Error: %v", err)
	}
	content, readErr := os.ReadFile(filepath.Join(logDir, "reports", "smoke.txt"))
	if readErr == nil {
		log("File content: %q", strings.TrimSpace(string(content)))
		rc.expectStatus = "completed" // validate file exists
	} else {
		log("File read error: %v", readErr)
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdValidateFail(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "validate-fail-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "validate-fail.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	baseRuntime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	validating := agentwrap.ValidatingRuntime{
		Runtime: baseRuntime,
		Spec: agentwrap.ValidationSpec{
			Expectations: []agentwrap.ValidationExpectation{
				{
					ID:       "required-file",
					Kind:     agentwrap.ExpectationFile,
					Path:     "reports/missing.txt",
					Severity: agentwrap.ExpectationRequired,
				},
			},
		},
	}

	var result agentwrap.RunResult
	run, err := validating.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Do not create any files. Reply only: done",
		WorkDir:  logDir,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
	} else {
		result, err = run.Wait(context.Background())
		log("Status: %s", result.Status)
		if err != nil {
			var sdkErr *agentwrap.SDKError
			if errors.As(err, &sdkErr) {
				log("Category: %s", sdkErr.Category)
			}
		}
		log("Validation failures: %d", len(result.Metadata.Validation.Final.Failures))
		for _, f := range result.Metadata.Validation.Final.Failures {
			log("  - %s: expected=%s observed=%s", f.ExpectationID, f.Expected, f.Observed)
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdValidateRepair(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "validate-repair-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(filepath.Join(logDir, "reports"), 0755)
	expectedFile := filepath.Join(logDir, "reports", "test.txt")
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "validate-repair.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	baseRuntime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	validating := agentwrap.ValidatingRuntime{
		Runtime: baseRuntime,
		Spec: agentwrap.ValidationSpec{
			Expectations: []agentwrap.ValidationExpectation{
				{
					ID:       "required-file",
					Kind:     agentwrap.ExpectationFile,
					Path:     "reports/test.txt",
					Severity: agentwrap.ExpectationRequired,
				},
			},
			Repair: agentwrap.RepairConfig{
				MaxAttempts:   2,
				SessionAction: agentwrap.SessionActionFresh,
				ShouldRepair:  func(ctx agentwrap.RepairContext) bool { return true },
				BuildPrompt: func(ctx agentwrap.RepairContext) string {
					return strings.Join([]string{
						"Create the file reports/test.txt in the current working directory.",
						"The file must contain exactly this single line:",
						"PASSED",
						"Do not write any other files or text.",
					}, "\n")
				},
			},
		},
	}

	var result agentwrap.RunResult
	run, err := validating.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Reply with exactly: initial validation should fail. Do not create files.",
		WorkDir:  logDir,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  90 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
	} else {
		result, err = run.Wait(context.Background())
		log("Status: %s", result.Status)
		if err != nil {
			var sdkErr *agentwrap.SDKError
			if errors.As(err, &sdkErr) {
				log("Category: %s", sdkErr.Category)
			}
		}
		log("Repair attempts: %d", len(result.Metadata.Repair.Attempts))
		for i, a := range result.Metadata.Repair.Attempts {
			log("  Repair %d: status=%s run=%s", i+1, a.Status, a.RunID)
		}
		content, _ := os.ReadFile(expectedFile)
		if content != nil {
			log("File content: %q", strings.TrimSpace(string(content)))
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdValidateRepairExhaust(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "validate-repair-exhaust-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "validate-repair-exhaust.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	baseRuntime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	impossibleValidator := agentwrap.ValidatorFunc(func(ctx context.Context, vctx agentwrap.ValidationContext) agentwrap.ValidationCheck {
		return agentwrap.ValidationCheck{
			ExpectationID: "impossible",
			Detail:        "always fails",
			Passed:        false,
		}
	})

	validating := agentwrap.ValidatingRuntime{
		Runtime: baseRuntime,
		Spec: agentwrap.ValidationSpec{
			Validators: []agentwrap.Validator{impossibleValidator},
			Repair: agentwrap.RepairConfig{
				MaxAttempts:   1,
				SessionAction: agentwrap.SessionActionFresh,
				ShouldRepair:  func(ctx agentwrap.RepairContext) bool { return true },
				BuildPrompt: func(ctx agentwrap.RepairContext) string {
					return "Reply with exactly: hello"
				},
			},
		},
	}

	var result agentwrap.RunResult
	run, err := validating.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Reply with exactly: hello",
		WorkDir:  logDir,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  90 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
	} else {
		result, err = run.Wait(context.Background())
		log("Status: %s", result.Status)
		if err != nil {
			var sdkErr *agentwrap.SDKError
			if errors.As(err, &sdkErr) {
				log("Category: %s", sdkErr.Category)
			}
		}
		log("Repair exhausted: %v", result.Metadata.Repair.Exhausted)
		log("Repair attempts: %d", len(result.Metadata.Repair.Attempts))
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdFallbackInvalidModel(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	primaryModel, _ := cmd.Flags().GetString("primary-model")
	fallbackModel, _ := cmd.Flags().GetString("fallback-model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "fallback-model-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "fallback.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	primary := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	fallback := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	runner := agentwrap.PolicyRunner{
		Runtime: primary,
		Policy: agentwrap.BasicPolicy{
			MaxAttemptsPerTarget: 1,
			RetryRateLimits:      true,
		},
		Alternatives: []agentwrap.FallbackAlternative{
			{
				Name:    "fallback",
				Runtime: fallback,
				Request: agentwrap.RunRequest{
					Provider: agentwrap.ProviderID("opencode"),
					Model:    agentwrap.ModelID(fallbackModel),
				},
			},
		},
	}

	log("Primary: %s, Fallback: %s", primaryModel, fallbackModel)
	run, err := runner.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello in 3 words.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(primaryModel),
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
		}
	}
	for _, a := range result.Metadata.Attempts {
		log("Attempt %d target=%d model=%s status=%s error=%s", a.Attempt, a.TargetIndex, a.Request.Model, a.Status, a.ErrorCategory)
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdFallbackInvalidProvider(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	primaryModel, _ := cmd.Flags().GetString("primary-model")
	fallbackModel, _ := cmd.Flags().GetString("fallback-model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "fallback-provider-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "fallback.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	primary := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	fallback := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	runner := agentwrap.PolicyRunner{
		Runtime: primary,
		Policy: agentwrap.BasicPolicy{
			MaxAttemptsPerTarget: 1,
			RetryRateLimits:      true,
		},
		Alternatives: []agentwrap.FallbackAlternative{
			{
				Name:    "fallback",
				Runtime: fallback,
				Request: agentwrap.RunRequest{
					Provider: agentwrap.ProviderID("opencode"),
					Model:    agentwrap.ModelID(fallbackModel),
				},
			},
		},
	}

	var result agentwrap.RunResult
	log("Primary: %s, Fallback: %s", primaryModel, fallbackModel)
	run, err := runner.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello in 3 words.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(primaryModel),
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
	} else {
		result, err = run.Wait(context.Background())
		log("Status: %s", result.Status)
		if err != nil {
			var sdkErr *agentwrap.SDKError
			if errors.As(err, &sdkErr) {
				log("Category: %s", sdkErr.Category)
			}
		}
		for _, a := range result.Metadata.Attempts {
			log("Attempt %d target=%d model=%s status=%s error=%s", a.Attempt, a.TargetIndex, a.Request.Model, a.Status, a.ErrorCategory)
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdFallbackAllFail(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "fallback-all-fail-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir, expectStatus: "failed", expectCategory: "model_unavailable"}

	logFile := filepath.Join(logDir, "fallback.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	primary := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	fallback := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	runner := agentwrap.PolicyRunner{
		Runtime: primary,
		Policy: agentwrap.BasicPolicy{
			MaxAttemptsPerTarget: 1,
			RetryRateLimits:      true,
		},
		Alternatives: []agentwrap.FallbackAlternative{
			{
				Name:    "bad-fallback",
				Runtime: fallback,
				Request: agentwrap.RunRequest{
					Provider: agentwrap.ProviderID("opencode"),
					Model:    agentwrap.ModelID("opencode/also-not-a-model"),
				},
			},
		},
	}

	log("Both primary and fallback use invalid models")
	run, err := runner.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello in 3 words.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID("opencode/not-a-real-model"),
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
	} else {
		result, err := run.Wait(context.Background())
		log("Status: %s", result.Status)
		if err != nil {
			var sdkErr *agentwrap.SDKError
			if errors.As(err, &sdkErr) {
				log("Category: %s", sdkErr.Category)
			}
		}
		for _, a := range result.Metadata.Attempts {
			log("Attempt %d target=%d model=%s status=%s error=%s", a.Attempt, a.TargetIndex, a.Request.Model, a.Status, a.ErrorCategory)
		}
		if sr := checkExpectation(rc, result, err); !sr.Passed {
			log("Expectation failed: got status=%s category=%s want status=%s category=%s", sr.Status, sr.Category, rc.expectStatus, rc.expectCategory)
		}
		saveResults(rc, time.Now(), result, err)
	}
}

func cmdInvalidSeparateProvider(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "invalid-separate-provider-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir, expectStatus: "failed", expectCategory: "configuration"}

	logFile := filepath.Join(logDir, "invalid-separate-provider.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	runtime := opencode.NewRuntime()
	log("Provider: bad/provider Model: test-model")
	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello in 3 words.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("bad/provider"),
		Model:    agentwrap.ModelID("test-model"),
		Timeout:  30 * time.Second,
	})
	var result agentwrap.RunResult
	if err != nil {
		log("StartRun error: %v", err)
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
	} else {
		result, err = run.Wait(context.Background())
		log("Status: %s", result.Status)
	}
	if sr := checkExpectation(rc, result, err); !sr.Passed {
		log("Expectation failed: got status=%s category=%s want status=%s category=%s", sr.Status, sr.Category, rc.expectStatus, rc.expectCategory)
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdHealthModel(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "health-model-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "health-model.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	report, err := runtime.CheckHealth(context.Background(), agentwrap.HealthCheckRequest{
		Context: agentwrap.RuntimeContext{
			RuntimeKind: agentwrap.RuntimeKind("opencode"),
			Provider:    agentwrap.ProviderID("opencode"),
			Model:       agentwrap.ModelID(model),
		},
		WorkDir: UltraPlanRoot,
		Checks: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckRuntimeAvailable,
			agentwrap.HealthCheckModel,
		},
		RequiredChecks: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckRuntimeAvailable,
		},
	})
	if err != nil {
		log("Health check error: %v", err)
	}
	log("Overall status: %s", report.OverallStatus)
	for _, r := range report.Results {
		log("  %s: %s - %s", r.Check, r.Status, r.UserDetail)
	}

	saveResults(&runConfig{logDir: logDir}, time.Now(), agentwrap.RunResult{}, nil)
}

func cmdVerifyEvidence(cmd *cobra.Command, args []string) {
	logDir := args[0]
	strict, _ := cmd.Flags().GetBool("strict")

	type evidenceIssue struct {
		Scenario string `json:"scenario"`
		File     string `json:"file,omitempty"`
		Field    string `json:"field,omitempty"`
		Issue    string `json:"issue"`
		Severity string `json:"severity"`
	}

	var allIssues []evidenceIssue
	scenarios := []string{}

	// Collect scenario subdirectories
	entries, err := os.ReadDir(logDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading log dir: %v\n", err)
		os.Exit(1)
	}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			scenarios = append(scenarios, e.Name())
		}
	}

	fmt.Printf("=== VERIFYING EVIDENCE: %s ===\n\n", logDir)
	fmt.Printf("Found %d scenarios\n\n", len(scenarios))

	for _, scenario := range scenarios {
		scDir := filepath.Join(logDir, scenario)
		fmt.Printf("--- %s ---\n", scenario)

		// 1. Check results.json exists and has required fields
		resultsFile := filepath.Join(scDir, "results.json")
		resultsData, err := os.ReadFile(resultsFile)
		if err != nil {
			issue := evidenceIssue{Scenario: scenario, File: "results.json", Issue: "missing", Severity: "error"}
			allIssues = append(allIssues, issue)
			fmt.Printf("  ERROR: results.json missing\n")
			continue
		}

		var results struct {
			Status      string         `json:"status"`
			SessionID   string         `json:"session_id"`
			Category    string         `json:"error_category,omitempty"`
			Error       string         `json:"error,omitempty"`
			NativeMeta  map[string]any `json:"native_metadata,omitempty"`
			CleanupMeta *struct {
				Attempted bool `json:"attempted"`
				Completed bool `json:"completed"`
				Failed    bool `json:"failed"`
			} `json:"cleanup_metadata,omitempty"`
			ValidationMeta *struct {
				Configured bool `json:"configured"`
			} `json:"validation_metadata,omitempty"`
			RepairMeta *struct {
				Configured bool `json:"configured"`
			} `json:"repair_metadata,omitempty"`
		}
		if err := json.Unmarshal(resultsData, &results); err != nil {
			issue := evidenceIssue{Scenario: scenario, File: "results.json", Issue: "invalid JSON: " + err.Error(), Severity: "error"}
			allIssues = append(allIssues, issue)
			fmt.Printf("  ERROR: results.json invalid JSON: %v\n", err)
			continue
		}

		// Check required fields in results.json
		if results.Status == "" {
			issue := evidenceIssue{Scenario: scenario, Field: "status", Issue: "missing or empty", Severity: "error"}
			allIssues = append(allIssues, issue)
			fmt.Printf("  ERROR: results.json missing status field\n")
		} else {
			fmt.Printf("  status: %s\n", results.Status)
		}

		if results.Status == "failed" && results.Category == "" {
			issue := evidenceIssue{Scenario: scenario, Field: "error_category", Issue: "missing for failed result", Severity: "warn"}
			allIssues = append(allIssues, issue)
			fmt.Printf("  WARN: failed result missing error_category\n")
		} else if results.Category != "" {
			fmt.Printf("  category: %s\n", results.Category)
		}

		// Check native_metadata present
		if results.NativeMeta == nil {
			issue := evidenceIssue{Scenario: scenario, Field: "native_metadata", Issue: "missing", Severity: "warn"}
			allIssues = append(allIssues, issue)
			fmt.Printf("  WARN: native_metadata missing\n")
		} else {
			fmt.Printf("  native_metadata: present (%d keys)\n", len(results.NativeMeta))
		}

		// Check cleanup_metadata present
		if results.CleanupMeta == nil {
			issue := evidenceIssue{Scenario: scenario, Field: "cleanup_metadata", Issue: "missing", Severity: "warn"}
			allIssues = append(allIssues, issue)
			fmt.Printf("  WARN: cleanup_metadata missing\n")
		} else {
			fmt.Printf("  cleanup: attempted=%v completed=%v failed=%v\n", results.CleanupMeta.Attempted, results.CleanupMeta.Completed, results.CleanupMeta.Failed)
		}

		// Check validation_metadata when configured
		if results.ValidationMeta != nil && results.ValidationMeta.Configured {
			json.Unmarshal(resultsData, &struct {
				ValidationMeta *struct {
					Configured bool `json:"configured"`
					Final      *struct {
						Passed bool `json:"passed"`
					} `json:"final"`
				} `json:"validation_metadata,omitempty"`
			}{&struct {
				Configured bool `json:"configured"`
				Final      *struct {
					Passed bool `json:"passed"`
				} `json:"final"`
			}{results.ValidationMeta.Configured, nil}})
			// Re-parse to check validation fields
			var fullResults struct {
				ValidationMeta *struct {
					Configured bool `json:"configured"`
					Final      *struct {
						Passed      bool  `json:"passed"`
						FailedCount int   `json:"failed_count"`
						Errors      []any `json:"errors"`
					} `json:"final"`
				} `json:"validation_metadata,omitempty"`
			}
			json.Unmarshal(resultsData, &fullResults)
			if fullResults.ValidationMeta == nil || fullResults.ValidationMeta.Final == nil {
				issue := evidenceIssue{Scenario: scenario, Field: "validation_metadata.final", Issue: "missing when validation configured", Severity: "warn"}
				allIssues = append(allIssues, issue)
				fmt.Printf("  WARN: validation_metadata.final missing when validation configured\n")
			} else {
				fmt.Printf("  validation: configured=true passed=%v failed_count=%d\n", fullResults.ValidationMeta.Final.Passed, fullResults.ValidationMeta.Final.FailedCount)
			}
		}

		// Check repair_metadata when configured
		if results.RepairMeta != nil && results.RepairMeta.Configured {
			var fullResults struct {
				RepairMeta *struct {
					Configured   bool   `json:"configured"`
					Exhausted    bool   `json:"exhausted"`
					Phase        string `json:"phase"`
					FailurePhase string `json:"failure_phase"`
				} `json:"repair_metadata,omitempty"`
			}
			json.Unmarshal(resultsData, &fullResults)
			if fullResults.RepairMeta == nil || fullResults.RepairMeta.Phase == "" {
				issue := evidenceIssue{Scenario: scenario, Field: "repair_metadata.phase", Issue: "missing when repair configured", Severity: "warn"}
				allIssues = append(allIssues, issue)
				fmt.Printf("  WARN: repair_metadata.phase missing when repair configured\n")
			} else {
				fmt.Printf("  repair: configured=true exhausted=%v phase=%s failure_phase=%s\n", fullResults.RepairMeta.Exhausted, fullResults.RepairMeta.Phase, fullResults.RepairMeta.FailurePhase)
			}
		}

		// 2. Check events.jsonl - only expected for real runs that emit events
		// Most smoke scenarios except health-fail, health-model emit events
		eventsFile := filepath.Join(scDir, "events.jsonl")
		if _, err := os.Stat(eventsFile); os.IsNotExist(err) {
			// events.jsonl is optional - only warn for scenarios that should have it
			if scenario != "health-fail" && scenario != "health-model" && scenario != "session-fork" {
				issue := evidenceIssue{Scenario: scenario, File: "events.jsonl", Issue: "missing (optional but expected for real runs)", Severity: "info"}
				allIssues = append(allIssues, issue)
				fmt.Printf("  INFO: events.jsonl missing (optional)\n")
			}
		} else {
			// Check it's not empty
			data, _ := os.ReadFile(eventsFile)
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			eventCount := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					eventCount++
				}
			}
			fmt.Printf("  events.jsonl: present (%d events)\n", eventCount)
		}

		// 3. Check opencode-db-snapshot-status.json exists
		dbStatusFile := filepath.Join(scDir, "opencode-db-snapshot-status.json")
		if _, err := os.Stat(dbStatusFile); os.IsNotExist(err) {
			issue := evidenceIssue{Scenario: scenario, File: "opencode-db-snapshot-status.json", Issue: "missing", Severity: "error"}
			allIssues = append(allIssues, issue)
			fmt.Printf("  ERROR: opencode-db-snapshot-status.json missing\n")
		} else {
			var dbStatus struct {
				SessionID string            `json:"session_id"`
				Status    string            `json:"status"`
				Files     map[string]string `json:"files,omitempty"`
				Errors    map[string]string `json:"errors,omitempty"`
			}
			if err := json.Unmarshal([]byte(readFileOrBlank(dbStatusFile)), &dbStatus); err != nil {
				issue := evidenceIssue{Scenario: scenario, File: "opencode-db-snapshot-status.json", Issue: "invalid JSON", Severity: "error"}
				allIssues = append(allIssues, issue)
				fmt.Printf("  ERROR: opencode-db-snapshot-status.json invalid JSON\n")
			} else {
				fmt.Printf("  db_snapshot: status=%s session_id=%s\n", dbStatus.Status, dbStatus.SessionID)

				// 4. If session ID exists, DB snapshot should be captured or failed
				if results.SessionID != "" {
					if dbStatus.Status != "captured" && dbStatus.Status != "failed" {
						issue := evidenceIssue{Scenario: scenario, Field: "opencode-db-snapshot-status.status", Issue: fmt.Sprintf("expected captured or failed, got %s", dbStatus.Status), Severity: "error"}
						allIssues = append(allIssues, issue)
						fmt.Printf("  ERROR: DB snapshot status=%s (expected captured or failed)\n", dbStatus.Status)
					}
					if dbStatus.SessionID != results.SessionID {
						issue := evidenceIssue{Scenario: scenario, Field: "opencode-db-snapshot-status.session_id", Issue: "session ID mismatch", Severity: "warn"}
						allIssues = append(allIssues, issue)
						fmt.Printf("  WARN: DB snapshot session_id=%s != result session_id=%s\n", dbStatus.SessionID, results.SessionID)
					}
				}

				// 5. If no session ID, DB snapshot should be skipped with reason
				if results.SessionID == "" {
					if dbStatus.Status != "skipped" {
						issue := evidenceIssue{Scenario: scenario, Field: "opencode-db-snapshot-status.status", Issue: fmt.Sprintf("expected skipped for no-session case, got %s", dbStatus.Status), Severity: "warn"}
						allIssues = append(allIssues, issue)
						fmt.Printf("  WARN: DB snapshot status=%s (expected skipped for no-session)\n", dbStatus.Status)
					}
					if dbStatus.Errors == nil || dbStatus.Errors["session"] == "" {
						issue := evidenceIssue{Scenario: scenario, Field: "opencode-db-snapshot-status.errors.session", Issue: "expected 'no session id' error reason", Severity: "warn"}
						allIssues = append(allIssues, issue)
						fmt.Printf("  WARN: DB snapshot errors.session missing 'no session id' reason\n")
					}
				}
			}
		}
		fmt.Println()
	}

	// Print summary
	fmt.Println("=== SUMMARY ===")
	if len(allIssues) == 0 {
		fmt.Println("No evidence issues found")
	} else {
		errorCount := 0
		warnCount := 0
		infoCount := 0
		for _, issue := range allIssues {
			switch issue.Severity {
			case "error":
				errorCount++
			case "warn":
				warnCount++
			case "info":
				infoCount++
			}
		}
		fmt.Printf("Issues: %d errors, %d warnings, %d info\n", errorCount, warnCount, infoCount)
		fmt.Println()

		if errorCount > 0 {
			fmt.Println("ERRORS:")
			for _, issue := range allIssues {
				if issue.Severity == "error" {
					fmt.Printf("  [%s] %s %s: %s\n", issue.Scenario, issue.File, issue.Field, issue.Issue)
				}
			}
			fmt.Println()
		}
		if warnCount > 0 {
			fmt.Println("WARNINGS:")
			for _, issue := range allIssues {
				if issue.Severity == "warn" {
					fmt.Printf("  [%s] %s %s: %s\n", issue.Scenario, issue.File, issue.Field, issue.Issue)
				}
			}
			fmt.Println()
		}
		if infoCount > 0 {
			fmt.Println("INFO:")
			for _, issue := range allIssues {
				if issue.Severity == "info" {
					fmt.Printf("  [%s] %s %s: %s\n", issue.Scenario, issue.File, issue.Field, issue.Issue)
				}
			}
			fmt.Println()
		}
	}

	if strict && len(allIssues) > 0 {
		fmt.Println("STRICT MODE: exiting with failure due to evidence issues")
		os.Exit(1)
	}
}

func readFileOrBlank(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// cmdInvalidProvider tests error category when provider is invalid
// Expected category: configuration (provider does not exist)
func cmdInvalidProvider(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "invalid-provider-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "invalid-provider.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	model := "nonexistent-provider/test"
	if modelStr != "" {
		model = modelStr
	}
	log("Model: %s", model)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello in 3 words.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
		os.Exit(1)
	}
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

// cmdInvalidModel tests error category when model is invalid
// Expected category: model_unavailable (model does not exist)
func cmdInvalidModel(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "invalid-model-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "invalid-model.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	model := "opencode/not-a-real-model"
	if modelStr != "" {
		model = modelStr
	}
	log("Model: %s", model)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello in 3 words.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
		os.Exit(1)
	}
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

// cmdOpenCodeDBUnavailable tests error category when OpenCode DB is unavailable
// Expected category: runtime_unavailable (DB checkpoint or lock failure)
// This is triggered by OpenCode's WAL checkpoint failures or database locked errors
func cmdOpenCodeDBUnavailable(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "opencode-db-unavailable-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "opencode-db-unavailable.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")

	// Create a scenario that triggers OpenCode DB issues by using a locked DB state
	// We simulate this by running a prompt that may trigger DB issues
	// In practice, this is hard to reproduce deterministically without patching OpenCode
	log("Testing OpenCode DB unavailable scenario")
	log("Note: This category is triggered by 'pragma wal_checkpoint', 'database is locked', or 'sqlite.*database' in OpenCode logs")

	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	// Try a simple run - if OpenCode has DB issues, the error classification will catch it
	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID("opencode/deepseek-v4-flash-free"),
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
		}
		os.Exit(1)
	}
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
			if sdkErr.Category == agentwrap.ErrorRuntimeUnavailable {
				log("OpenCode DB issue detected correctly")
			}
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdDBUnavailable(cmd *cobra.Command, args []string) {
	cmdFakeDBBoundary(cmd, "db-unavailable", "unavailable", "partial")
}

func cmdDBNonJSON(cmd *cobra.Command, args []string) {
	cmdFakeDBBoundary(cmd, "db-non-json", "nonjson", "partial")
}

func cmdDBTimeout(cmd *cobra.Command, args []string) {
	cmdFakeDBBoundary(cmd, "db-timeout", "timeout", "partial")
}

func cmdDBLocked(cmd *cobra.Command, args []string) {
	cmdFakeDBBoundary(cmd, "db-locked", "locked", "partial")
}

func cmdDBSessionNoAssistant(cmd *cobra.Command, args []string) {
	cmdFakeDBBoundary(cmd, "db-session-no-assistant", "session_no_assistant", "final")
}

func cmdDBAssistantNoFinish(cmd *cobra.Command, args []string) {
	cmdFakeDBBoundary(cmd, "db-assistant-no-finish", "assistant_no_finish", "final")
}

func cmdDBAssistantWithFinish(cmd *cobra.Command, args []string) {
	cmdFakeDBBoundary(cmd, "db-assistant-with-finish", "complete", "final")
}

func cmdDBOnlyProof(cmd *cobra.Command, args []string) {
	cmdFakeDBBoundary(cmd, "db-only-proof", "db_only_proof", "db_proof")
}

func cmdFakeDBBoundary(cmd *cobra.Command, scenario, dbMode, runMode string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", scenario+"-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, scenario+".log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	fakeOpenCode := filepath.Join(UltraPlanRoot, "testdata", "fakeopencode", "fake-opencode.sh")
	rc.dbExecutable = fakeOpenCode
	rc.dbEnv = []string{"DB_MODE=" + dbMode, "RUN_MODE=" + runMode}
	log("Fake OpenCode: %s", fakeOpenCode)
	log("RUN_MODE=%s DB_MODE=%s", runMode, dbMode)
	runtime := opencode.NewRuntime(
		opencode.WithExecutable(fakeOpenCode),
		opencode.WithEnv("RUN_MODE="+runMode, "DB_MODE="+dbMode),
	)
	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:  "Say hello.",
		WorkDir: UltraPlanRoot,
		Model:   agentwrap.ModelID("opencode/fake"),
		Timeout: 2 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		saveResults(rc, time.Now(), agentwrap.RunResult{Status: agentwrap.StatusFailed}, err)
		saveOpenCodeDB(rc, "")
		return
	}
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

// cmdProvider429 tests error category when provider returns 429
// Expected category: rate_limit
// Note: This depends on a real provider 429 occurring, which is non-deterministic
// The smoke test documents the expected category when it occurs
func cmdProvider429(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "provider-429-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "provider-429.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}
	log("Model: %s", model)
	log("Note: Provider 429 is non-deterministic; expected category is rate_limit when it occurs")

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello in 3 words.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			if sdkErr.Category == agentwrap.ErrorRateLimit {
				log("Rate limit detected correctly")
			}
		}
	}
	for _, a := range result.Metadata.Attempts {
		if a.RateLimit != nil {
			log("RateLimit on attempt %d: provider=%s model=%s detail=%s",
				a.Attempt, a.RateLimit.Provider, a.RateLimit.Model, a.RateLimit.UserDetail)
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

// cmdProviderAuth tests error category when provider has auth/account issues
// Expected category: authentication
// Note: This depends on provider authentication failures occurring
func cmdProviderAuth(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "provider-auth-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "provider-auth.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}
	log("Model: %s", model)
	log("Note: Provider auth issues are non-deterministic; expected category is authentication when they occur")

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello in 3 words.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			if sdkErr.Category == agentwrap.ErrorAuthentication {
				log("Authentication error detected correctly")
			}
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

// cmdFatalClean tests error category when OpenCode emits a clean fatal event
// with no provider-specific detail
// Expected category: runtime_exit
// This is the current behavior: fatal events without rate-limit classification
// become runtime_exit
func cmdFatalClean(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "fatal-clean-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "fatal-clean.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	// Use a model that will cause OpenCode to emit a fatal error event
	// "opencode/not-a-real-model" triggers "Model not found" fatal event
	model := "opencode/not-a-real-model"
	log("Model: %s (expected to cause fatal event)", model)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Say hello.",
		WorkDir:  UltraPlanRoot,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
			log("DebugDetail: %s", sdkErr.DebugDetail)
			// Verify this is runtime_exit (current behavior for clean fatal events)
			if sdkErr.Category == agentwrap.ErrorRuntimeExit {
				log("Confirmed: clean fatal event -> runtime_exit")
			}
		}
	}
	// Log event categories to understand what OpenCode emitted
	for k, v := range result.Metadata.NativeMetadata {
		if k == "event_categories" || k == "native_event_types" {
			log("Native metadata %s: %v", k, v)
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdSessionFresh(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "session-fresh-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "session-fresh.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== SESSION LIFECYCLE: FRESH SESSION SUCCESS ===")
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:        "Reply with exactly: OK",
		WorkDir:       UltraPlanRoot,
		Provider:      agentwrap.ProviderID("opencode"),
		Model:         agentwrap.ModelID(model),
		Timeout:       30 * time.Second,
		WantSession:   true,
		SessionAction: agentwrap.SessionActionFresh,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())

	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if result.SessionID != "" {
		log("Session ID: %s", result.SessionID)
	}
	if result.Metadata.Session.ID != "" {
		log("Session metadata ID: %s", result.Metadata.Session.ID)
		log("Session relationship: %s", result.Metadata.Session.Relationship)
		log("Session requested action: %s", result.Metadata.Session.RequestedAction)
	}
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
		}
	}

	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

func cmdSessionContinueExisting(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "session-continue-existing-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "session-continue-existing.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== SESSION LIFECYCLE: CONTINUE EXISTING SESSION SUCCESS ===")
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	run1, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:        "Reply with exactly: FIRST",
		WorkDir:       UltraPlanRoot,
		Provider:      agentwrap.ProviderID("opencode"),
		Model:         agentwrap.ModelID(model),
		Timeout:       30 * time.Second,
		WantSession:   true,
		SessionAction: agentwrap.SessionActionFresh,
	})
	if err != nil {
		log("StartRun 1 error: %v", err)
		os.Exit(1)
	}
	result1, err := run1.Wait(context.Background())
	log("Run 1 Status: %s", result1.Status)
	if result1.SessionID != "" {
		log("Run 1 Session ID: %s", result1.SessionID)
	} else {
		log("Run 1: No session ID returned")
	}

	if result1.SessionID == "" {
		log("ERROR: Run 1 did not return a session ID, cannot continue")
		saveResults(rc, time.Now(), result1, err)
		os.Exit(1)
	}

	sessionID := result1.SessionID
	log("")
	log("Attempting to continue session %s", sessionID)

	run2, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:        "Reply with exactly: SECOND",
		WorkDir:       UltraPlanRoot,
		Provider:      agentwrap.ProviderID("opencode"),
		Model:         agentwrap.ModelID(model),
		Timeout:       30 * time.Second,
		WantSession:   true,
		SessionAction: agentwrap.SessionActionContinue,
		SessionID:     agentwrap.SessionID(sessionID),
	})
	if err != nil {
		log("StartRun 2 error: %v", err)
		os.Exit(1)
	}
	log("Run 2 started: ID=%s", run2.ID())

	result2, err := run2.Wait(context.Background())
	log("Run 2 Status: %s", result2.Status)
	if result2.SessionID != "" {
		log("Run 2 Session ID: %s", result2.SessionID)
	}
	if result2.Metadata.Session.ID != "" {
		log("Session metadata ID: %s", result2.Metadata.Session.ID)
		log("Session relationship: %s", result2.Metadata.Session.Relationship)
		log("Session requested action: %s", result2.Metadata.Session.RequestedAction)
		log("Session continued: %v", result2.Metadata.Session.Continued)
	}
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
		}
	}

	saveResults(rc, time.Now(), result2, err)
	saveOpenCodeDB(rc, string(result2.SessionID))
}

func cmdSessionContinueMissing(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "session-continue-missing-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir, expectStatus: "failed", expectCategory: "runtime_exit"}

	logFile := filepath.Join(logDir, "session-continue-missing.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== SESSION LIFECYCLE: CONTINUE MISSING/INVALID SESSION ===")
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	missingSessionID := "ses_does_not_exist_12345"
	log("Attempting to continue non-existent session: %s", missingSessionID)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:        "Reply with exactly: OK",
		WorkDir:       UltraPlanRoot,
		Provider:      agentwrap.ProviderID("opencode"),
		Model:         agentwrap.ModelID(model),
		Timeout:       30 * time.Second,
		WantSession:   true,
		SessionAction: agentwrap.SessionActionContinue,
		SessionID:     agentwrap.SessionID(missingSessionID),
	})
	if err != nil {
		log("StartRun error: %v", err)
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())

	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if result.SessionID != "" {
		log("Session ID: %s", result.SessionID)
	}
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
	} else {
		log("No error returned - continue with missing session may silently start fresh")
	}

	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
	sr := checkExpectation(rc, result, err)
	log("Expectation: status=%s category=%s passed=%v", rc.expectStatus, rc.expectCategory, sr.Passed)
}

func cmdSessionContinueAfterFail(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "session-continue-after-fail-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "session-continue-after-fail.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== SESSION LIFECYCLE: CONTINUE AFTER FAILED RUN ===")
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	runtime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))

	run1, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:        "Do something that will timeout quickly",
		WorkDir:       UltraPlanRoot,
		Provider:      agentwrap.ProviderID("opencode"),
		Model:         agentwrap.ModelID(model),
		Timeout:       100 * time.Millisecond,
		WantSession:   true,
		SessionAction: agentwrap.SessionActionFresh,
	})
	if err != nil {
		log("StartRun 1 error: %v", err)
	}
	result1, err := run1.Wait(context.Background())
	log("Run 1 Status: %s", result1.Status)
	if result1.SessionID != "" {
		log("Run 1 Session ID: %s", result1.SessionID)
	}

	if result1.SessionID == "" {
		log("Run 1 produced no session - cannot test continue after fail")
		saveResults(rc, time.Now(), result1, err)
		os.Exit(1)
	}

	sessionID := result1.SessionID
	log("")
	log("Attempting to continue session %s after failed run", sessionID)

	run2, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:        "Reply with exactly: RECOVERED",
		WorkDir:       UltraPlanRoot,
		Provider:      agentwrap.ProviderID("opencode"),
		Model:         agentwrap.ModelID(model),
		Timeout:       30 * time.Second,
		WantSession:   true,
		SessionAction: agentwrap.SessionActionContinue,
		SessionID:     agentwrap.SessionID(sessionID),
	})
	if err != nil {
		log("StartRun 2 error: %v", err)
		os.Exit(1)
	}
	log("Run 2 started: ID=%s", run2.ID())

	result2, err := run2.Wait(context.Background())
	log("Run 2 Status: %s", result2.Status)
	if result2.SessionID != "" {
		log("Run 2 Session ID: %s", result2.SessionID)
	}
	if result2.Metadata.Session.ID != "" {
		log("Session relationship: %s", result2.Metadata.Session.Relationship)
		log("Session continued: %v", result2.Metadata.Session.Continued)
	}
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
		}
	}

	saveResults(rc, time.Now(), result2, err)
	saveOpenCodeDB(rc, string(result2.SessionID))
}

func cmdRepairWithContinue(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	modelStr, _ := cmd.Flags().GetString("model")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "repair-with-continue-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(filepath.Join(logDir, "reports"), 0755)
	rc := &runConfig{logDir: logDir}

	logFile := filepath.Join(logDir, "repair-with-continue.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== SESSION LIFECYCLE: REPAIR WITH SessionActionContinue ===")
	log("")

	cfgJSON, _ := config.Load(filepath.Join(UltraPlanRoot, "config.json"))
	opencodeConfigPath := filepath.Join(UltraPlanRoot, "cli", "opencode-config.json")
	model := cfgJSON.SprintExecutionModel
	if modelStr != "" {
		model = modelStr
	}

	baseRuntime := opencode.NewRuntime(opencode.WithEnv("OPENCODE_CONFIG=" + opencodeConfigPath))
	validating := agentwrap.ValidatingRuntime{
		Runtime: baseRuntime,
		Spec: agentwrap.ValidationSpec{
			Expectations: []agentwrap.ValidationExpectation{
				{
					ID:       "required-file",
					Kind:     agentwrap.ExpectationFile,
					Path:     "reports/test.txt",
					Severity: agentwrap.ExpectationRequired,
				},
			},
			Repair: agentwrap.RepairConfig{
				MaxAttempts:   1,
				SessionAction: agentwrap.SessionActionContinue,
				ShouldRepair:  func(ctx agentwrap.RepairContext) bool { return true },
				BuildPrompt: func(ctx agentwrap.RepairContext) string {
					return strings.Join([]string{
						"Create the file reports/test.txt in the current working directory.",
						"The file must contain exactly this single line:",
						"PASSED",
						"Do not write any other files or text.",
					}, "\n")
				},
			},
		},
	}

	log("ValidatingRuntime with Repair SessionAction=Continue")
	log("Using model: %s", model)

	var result agentwrap.RunResult
	run, err := validating.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:   "Reply with exactly: initial validation should fail. Do not create files.",
		WorkDir:  logDir,
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID(model),
		Timeout:  90 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
	} else {
		result, err = run.Wait(context.Background())
		log("Status: %s", result.Status)
		if err != nil {
			var sdkErr *agentwrap.SDKError
			if errors.As(err, &sdkErr) {
				log("Category: %s", sdkErr.Category)
			}
		}
		log("Repair attempts: %d", len(result.Metadata.Repair.Attempts))
		for i, a := range result.Metadata.Repair.Attempts {
			log("  Repair %d: status=%s run=%s", i+1, a.Status, a.RunID)
			log("  Repair session: relationship=%s continued=%v",
				a.Session.Relationship, a.Session.Continued)
		}
		if result.Metadata.Session.Relationship != "" {
			log("Final session relationship: %s", result.Metadata.Session.Relationship)
		}
	}
	saveResults(rc, time.Now(), result, err)
	saveOpenCodeDB(rc, string(result.SessionID))
}

// =============================================================================
// Workstream 7: Process-Group Cleanup and Final-State Precedence
// =============================================================================

// cmdProcessGroupNonZeroFinal tests Case 1: Non-zero exit with final event and no provider error.
// Expected: completed (final event supersedes exit code)
func cmdProcessGroupNonZeroFinal(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "process-group-nonzero-final-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir, expectStatus: "completed", expectCategory: ""}

	logFile := filepath.Join(logDir, "process-group-nonzero-final.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== PROCESS-GROUP TEST: Non-Zero Exit With Final Event ===")
	log("Expected: completed (final event supersedes exit code)")
	log("")

	fakeOpenCode := filepath.Join(UltraPlanRoot, "fake-opencode", "fake-opencode.sh")
	log("Using fake opencode: %s", fakeOpenCode)

	// RUN_MODE=nonzero_final: emits step_start, text, step_finish, then exits 7
	runtime := opencode.NewRuntime(
		opencode.WithExecutable(fakeOpenCode),
		opencode.WithEnv("RUN_MODE=nonzero_final"),
	)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:  "Say hello.",
		WorkDir: UltraPlanRoot,
		Model:   agentwrap.ModelID("opencode/fake"),
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())

	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
	} else {
		log("No error returned - run completed successfully")
	}

	// Log evidence
	for k, v := range result.Metadata.NativeMetadata {
		if k == "exit_code" {
			log("Native metadata %s: %v", k, v)
		}
	}

	saveResults(rc, time.Now(), result, err)
	sr := checkExpectation(rc, result, err)
	log("Expectation: status=%s category=%s passed=%v", rc.expectStatus, rc.expectCategory, sr.Passed)
}

// cmdProcessGroupNonZeroRateLimit tests Case 2: Non-zero exit with final event plus explicit rate-limit stderr.
// Expected: rate_limit (stderr contains rate-limit shape, metadata includes rate_limit_info)
func cmdProcessGroupNonZeroRateLimit(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "process-group-nonzero-ratelimit-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir, expectStatus: "failed", expectCategory: "rate_limit"}

	logFile := filepath.Join(logDir, "process-group-nonzero-ratelimit.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== PROCESS-GROUP TEST: Non-Zero Exit With Rate-Limit Stderr ===")
	log("Expected: failed/rate_limit (rate-limit stderr should classify as rate_limit)")
	log("")

	fakeOpenCode := filepath.Join(UltraPlanRoot, "fake-opencode", "fake-opencode.sh")
	log("Using fake opencode: %s", fakeOpenCode)

	// RUN_MODE=nonzero_ratelimit: emits final events, then rate-limit stderr, then exits 7
	runtime := opencode.NewRuntime(
		opencode.WithExecutable(fakeOpenCode),
		opencode.WithEnv("RUN_MODE=nonzero_ratelimit"),
	)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:  "Say hello.",
		WorkDir: UltraPlanRoot,
		Model:   agentwrap.ModelID("opencode/fake"),
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())

	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
	}

	// Log evidence
	for k, v := range result.Metadata.NativeMetadata {
		if k == "exit_code" || k == "rate_limit_info" || k == "stderr" {
			log("Native metadata %s: %v", k, v)
		}
	}

	saveResults(rc, time.Now(), result, err)
	sr := checkExpectation(rc, result, err)
	log("Expectation: status=%s category=%s passed=%v", rc.expectStatus, rc.expectCategory, sr.Passed)
}

// cmdProcessGroupCancelWithChildren tests Case 4: Cancellation should terminate the whole process group.
// Expected: cancelled (no surviving child processes after cancel)
func cmdProcessGroupCancelWithChildren(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "process-group-cancel-children-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir, expectStatus: "cancelled", expectCategory: "cancellation"}

	logFile := filepath.Join(logDir, "process-group-cancel-children.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== PROCESS-GROUP TEST: Cancellation With Child Processes ===")
	log("Expected: cancelled (whole process group terminated, no surviving children)")
	log("")

	fakeOpenCode := filepath.Join(UltraPlanRoot, "fake-opencode", "fake-opencode.sh")
	pidDir := filepath.Join(logDir, "pids")
	log("Using fake opencode: %s", fakeOpenCode)
	log("PID evidence dir: %s", pidDir)

	// HELPER_CHILD=1: spawns a helper that exits when parent receives SIGTERM
	runtime := opencode.NewRuntime(
		opencode.WithExecutable(fakeOpenCode),
		opencode.WithEnv("RUN_MODE=timeout", "HELPER_CHILD=1", "OPENCODE_PID_DIR="+pidDir),
	)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:  "Count to 100 slowly.",
		WorkDir: UltraPlanRoot,
		Model:   agentwrap.ModelID("opencode/fake"),
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())
	log("Waiting 2 seconds before cancelling...")
	time.Sleep(2 * time.Second)

	log("Calling Run.Cancel()...")
	cancelErr := run.Cancel(context.Background())
	if cancelErr != nil {
		log("Cancel error: %v", cancelErr)
	}

	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
		}
	}

	time.Sleep(300 * time.Millisecond)
	opencodePID := readPIDFile(filepath.Join(pidDir, "opencode.pid"))
	helperPID := readPIDFile(filepath.Join(pidDir, "helper.pid"))
	opencodeAlive := processAlive(opencodePID)
	helperAlive := processAlive(helperPID)
	log("Fake OpenCode PID: %d alive=%v", opencodePID, opencodeAlive)
	log("Helper PID: %d alive=%v", helperPID, helperAlive)
	if opencodeAlive || helperAlive {
		log("WARNING: process-group cleanup left survivor(s): opencode_alive=%v helper_alive=%v", opencodeAlive, helperAlive)
	}

	saveResults(rc, time.Now(), result, err)
	sr := checkExpectation(rc, result, err)
	log("Expectation: status=%s category=%s passed=%v", rc.expectStatus, rc.expectCategory, sr.Passed)
}

func readPIDFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return false
	}
	fields := strings.Fields(string(stat))
	if len(fields) >= 3 && fields[2] == "Z" {
		return false
	}
	return true
}

// cmdProcessGroupMalformedBeforeFinal tests Case 6: Malformed output before a final event should still fail.
// Expected: failed/malformed_event (malformed output before final event causes decode error)
func cmdProcessGroupMalformedBeforeFinal(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "process-group-malformed-before-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir, expectStatus: "failed", expectCategory: "malformed_event"}

	logFile := filepath.Join(logDir, "process-group-malformed-before.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== PROCESS-GROUP TEST: Malformed Output Before Final Event ===")
	log("Expected: failed/malformed_event (malformed output before final event causes failure)")
	log("")

	fakeOpenCode := filepath.Join(UltraPlanRoot, "fake-opencode", "fake-opencode.sh")
	log("Using fake opencode: %s", fakeOpenCode)

	// RUN_MODE=malformed_before_final: step_start, malformed JSON, text, step_finish
	runtime := opencode.NewRuntime(
		opencode.WithExecutable(fakeOpenCode),
		opencode.WithEnv("RUN_MODE=malformed_before_final"),
	)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:  "Say hello.",
		WorkDir: UltraPlanRoot,
		Model:   agentwrap.ModelID("opencode/fake"),
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())

	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
			log("UserDetail: %s", sdkErr.UserDetail)
		}
	}

	saveResults(rc, time.Now(), result, err)
	sr := checkExpectation(rc, result, err)
	log("Expectation: status=%s category=%s passed=%v", rc.expectStatus, rc.expectCategory, sr.Passed)
}

// cmdProcessGroupMalformedAfterFinal tests Case 6: Final event followed by malformed output should still succeed.
// Expected: completed (final event should not be lost because of malformed output after it)
func cmdProcessGroupMalformedAfterFinal(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "process-group-malformed-after-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir, expectStatus: "completed", expectCategory: ""}

	logFile := filepath.Join(logDir, "process-group-malformed-after.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== PROCESS-GROUP TEST: Final Event Followed By Malformed Output ===")
	log("Expected: completed (final event should win despite malformed output after)")
	log("")

	fakeOpenCode := filepath.Join(UltraPlanRoot, "fake-opencode", "fake-opencode.sh")
	log("Using fake opencode: %s", fakeOpenCode)

	// RUN_MODE=malformed_after_final: step_start, text, step_finish, malformed JSON
	runtime := opencode.NewRuntime(
		opencode.WithExecutable(fakeOpenCode),
		opencode.WithEnv("RUN_MODE=malformed_after_final"),
	)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:  "Say hello.",
		WorkDir: UltraPlanRoot,
		Model:   agentwrap.ModelID("opencode/fake"),
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())

	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
		}
	} else {
		log("No error returned - run completed successfully")
	}

	saveResults(rc, time.Now(), result, err)
	sr := checkExpectation(rc, result, err)
	log("Expectation: status=%s category=%s passed=%v", rc.expectStatus, rc.expectCategory, sr.Passed)
}

// cmdProcessGroupFinalDelayed tests Case 3: Final event after delayed child exit - event should win.
// Expected: completed (final event arrives before process termination and wins final classification)
func cmdProcessGroupFinalDelayed(cmd *cobra.Command, args []string) {
	logDir, _ := cmd.Flags().GetString("log-dir")
	if logDir == "" {
		logDir = filepath.Join(UltraPlanRoot, ".agentwrap-logs", "process-group-final-delayed-"+time.Now().Format("20060102-150405"))
	}
	os.MkdirAll(logDir, 0755)
	rc := &runConfig{logDir: logDir, expectStatus: "completed", expectCategory: ""}

	logFile := filepath.Join(logDir, "process-group-final-delayed.log")
	f, _ := os.Create(logFile)
	defer f.Close()
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		f.Write([]byte(msg + "\n"))
		fmt.Printf(format+"\n", args...)
	}

	log("=== PROCESS-GROUP TEST: Final Event After Delayed Exit ===")
	log("Expected: completed (final event arrives before process termination, wins classification)")
	log("")

	fakeOpenCode := filepath.Join(UltraPlanRoot, "fake-opencode", "fake-opencode.sh")
	log("Using fake opencode: %s", fakeOpenCode)

	// RUN_MODE=final_delayed: step_start, text, sleeps 1s, then step_finish
	runtime := opencode.NewRuntime(
		opencode.WithExecutable(fakeOpenCode),
		opencode.WithEnv("RUN_MODE=final_delayed"),
	)

	run, err := runtime.StartRun(context.Background(), agentwrap.RunRequest{
		Prompt:  "Say hello.",
		WorkDir: UltraPlanRoot,
		Model:   agentwrap.ModelID("opencode/fake"),
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log("StartRun error: %v", err)
		os.Exit(1)
	}
	log("Run started: ID=%s", run.ID())

	result, err := run.Wait(context.Background())
	log("Status: %s", result.Status)
	if err != nil {
		var sdkErr *agentwrap.SDKError
		if errors.As(err, &sdkErr) {
			log("Category: %s", sdkErr.Category)
		}
	} else {
		log("No error returned - run completed successfully")
	}

	saveResults(rc, time.Now(), result, err)
	sr := checkExpectation(rc, result, err)
	log("Expectation: status=%s category=%s passed=%v", rc.expectStatus, rc.expectCategory, sr.Passed)
}
