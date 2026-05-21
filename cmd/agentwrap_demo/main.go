package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/antonioborgerees/agentwrap"
	"github.com/antonioborgerees/agentwrap/opencode"
)

type metricsSink struct{}

func (s metricsSink) AppendEvent(ctx context.Context, rec agentwrap.RunEventRecord) error {
	return nil
}

func demoValidation() {
	fmt.Println("\n=== VALIDATION DEMO ===")
	fmt.Println("agentwrap can validate outputs after a run completes, with optional repair")

	root := os.Getenv("ULTRAPLAN_ROOT")
	if root == "" {
		root, _ = os.Getwd()
	}

	rt := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG=" + filepath.Join(root, "opencode-config.json")),
	)

	spec := agentwrap.ValidationSpec{
		Expectations: []agentwrap.ValidationExpectation{
			{
				ID:      "report-file",
				Kind:    agentwrap.ExpectationFile,
				Path:    "reports/final/test.md",
				Severity: agentwrap.ExpectationRequired,
			},
			{
				ID:           "report-template",
				Kind:         agentwrap.ExpectationMarkdownTemplate,
				Path:         "reports/final/test.md",
				TemplatePath: "template.md",
				Severity:     agentwrap.ExpectationOptional,
			},
			{
				ID:             "summary-json",
				Kind:           agentwrap.ExpectationJSON,
				Path:           "summary.json",
				RequiredFields: []string{"status", "score"},
				Severity:       agentwrap.ExpectationOptional,
			},
		},
		Validators: []agentwrap.Validator{
			agentwrap.ValidatorFunc(func(ctx context.Context, vctx agentwrap.ValidationContext) agentwrap.ValidationCheck {
				return agentwrap.ValidationCheck{
					ExpectationID: "custom-check",
					Kind:          agentwrap.ExpectationCustom,
					Passed:        true,
					Expected:      "custom validation passed",
					Detail:        "This is a caller-defined validator",
				}
			}),
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
	}

	validating := agentwrap.ValidatingRuntime{
		Runtime: rt,
		Spec:    spec,
	}

	fmt.Printf("  ValidationSpec: %d expectations + %d custom validators\n",
		len(spec.Expectations), len(spec.Validators))
	fmt.Printf("  ValidatingRuntime wraps: %T\n", validating.Runtime)
	fmt.Printf("  Repair: MaxAttempts=%d, SessionAction=%s\n",
		spec.Repair.MaxAttempts, spec.Repair.SessionAction)
}

func demoPolicy() {
	fmt.Println("\n=== RESILIENCE POLICY DEMO ===")
	fmt.Println("agentwrap provides PolicyRunner for bounded retry, backoff, and fallback")

	primary := opencode.NewRuntime()
	fallback := opencode.NewRuntime()

	policy := agentwrap.BasicPolicy{
		MaxAttemptsPerTarget: 3,
		MaxElapsed:           10 * time.Minute,
		Backoff: agentwrap.ExponentialBackoff{
			Initial: 1 * time.Second,
			Factor:  2,
			Max:     30 * time.Second,
		},
		RetryRateLimits: true,
		Fallbacks: []agentwrap.FallbackAlternative{
			{
				Name:    "claude-haiku",
				Runtime: fallback,
				Request: agentwrap.RunRequest{
					Provider: agentwrap.ProviderID("opencode"),
					Model:   agentwrap.ModelID("opencode/deepseek-v4-flash-free"),
				},
			},
		},
	}

	runner := agentwrap.PolicyRunner{
		Runtime:      primary,
		Alternatives: policy.Fallbacks,
		Policy:       policy,
	}

	_ = runner
	fmt.Printf("  BasicPolicy: MaxAttempts=%d, MaxElapsed=%v\n", policy.MaxAttemptsPerTarget, policy.MaxElapsed)
	if expB, ok := policy.Backoff.(agentwrap.ExponentialBackoff); ok {
		fmt.Printf("  Backoff: Initial=%v, Factor=%v, Max=%v\n", expB.Initial, expB.Factor, expB.Max)
	} else {
		fmt.Printf("  Backoff type: %T\n", policy.Backoff)
	}
	fmt.Printf("  RetryRateLimits: %v\n", policy.RetryRateLimits)
	fmt.Printf("  Fallbacks: %d alternatives\n", len(policy.Fallbacks))
	fmt.Println("  PolicyRunner wraps Runtime for automatic retry/fallback handling")
}

func demoObservability() {
	fmt.Println("\n=== OBSERVABILITY DEMO ===")
	fmt.Println("ObservingRuntime wraps any Runtime for event sinks and run store")

	store := agentwrap.NewMemoryRunStore()

	sink := metricsSink{}
	namedSink := agentwrap.NamedEventSink{
		Name:     "metrics",
		Sink:     sink,
		Required: false,
	}

	_ = namedSink

	fmt.Printf("  MemoryRunStore created: %T\n", store)
	fmt.Println("  ObservingRuntime wraps Runtime with Store + Sinks for observability")
	fmt.Println("  RunStore supports: UpsertRun, AppendEvent, ListActiveRuns, GetCompletedRun, ListRunEvents")
	fmt.Println("  EventSink: AppendEvent(ctx, RunEventRecord) for real-time event streaming")
}

func demoHealthChecks() {
	fmt.Println("\n=== HEALTH CHECKS DEMO ===")
	fmt.Println("Health check system for pre-flight validation")

	ctx := context.Background()
	rt := opencode.NewRuntime()

	report, err := rt.CheckHealth(ctx, agentwrap.HealthCheckRequest{
		Context: agentwrap.RuntimeContext{
			RuntimeKind: agentwrap.RuntimeKind("opencode"),
			RuntimeName: "opencode",
			Provider:    agentwrap.ProviderID("opencode"),
			Model:      agentwrap.ModelID("minimax-coding-plan/MiniMax-M2.7"),
		},
		WorkDir:  ".",
		Provider: "opencode",
		Model:    "minimax-coding-plan/MiniMax-M2.7",
		Timeout:  30 * time.Second,
		Checks: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckRuntimeAvailable,
			agentwrap.HealthCheckStructuredOutput,
			agentwrap.HealthCheckWorkDir,
			agentwrap.HealthCheckConfig,
			agentwrap.HealthCheckRuntimePaths,
			agentwrap.HealthCheckProvider,
			agentwrap.HealthCheckModel,
		},
		RequiredChecks: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckRuntimeAvailable,
		},
	})

	if err != nil {
		fmt.Printf("  Health check error: %v\n", err)
	} else {
		fmt.Printf("  HealthReport: OverallStatus=%s\n", report.OverallStatus)
		fmt.Printf("  %d health checks performed\n", len(report.Results))
		required := []agentwrap.HealthCheckID{agentwrap.HealthCheckRuntimeAvailable}
		if failure := agentwrap.RequiredHealthFailure(report, required); failure != nil {
			fmt.Printf("  Required check failure: %v\n", failure)
		}
	}
}

func demoPermissions() {
	fmt.Println("\n=== PERMISSIONS DEMO ===")
	fmt.Println("Structured permission policy with runtime-neutral tool classes")

	policy := agentwrap.PermissionPolicy{
		Default: agentwrap.PermissionActionDeny,
		Tools: map[agentwrap.PermissionTool]agentwrap.PermissionAction{
			agentwrap.PermissionToolRead:    agentwrap.PermissionActionAllow,
			agentwrap.PermissionToolEdit:   agentwrap.PermissionActionAllow,
			agentwrap.PermissionToolShell:  agentwrap.PermissionActionAsk,
			agentwrap.PermissionToolGlob:   agentwrap.PermissionActionAllow,
			agentwrap.PermissionToolSearch: agentwrap.PermissionActionAllow,
		},
		UnsupportedBehavior: agentwrap.PermissionUnsupportedBestEffort,
	}

	summary := policy.Summary()
	fmt.Printf("  Policy Default: %s\n", policy.Default)
	fmt.Printf("  Tool policies: %d tools configured\n", len(policy.Tools))
	fmt.Printf("  UnsupportedBehavior: %s\n", policy.UnsupportedBehavior)
	fmt.Printf("  Policy ID (SHA256): %s\n", summary.ID)

	err := agentwrap.ValidatePermissionPolicy(&policy)
	fmt.Printf("  ValidatePermissionPolicy: %v\n", err)
}

func demoCapabilities() {
	fmt.Println("\n=== CAPABILITIES DEMO ===")
	fmt.Println("Feature detection via Capabilities system")

	ctx := context.Background()
	rt := opencode.NewRuntime()

	caps, err := rt.Capabilities(ctx)
	if err != nil {
		fmt.Printf("  Capabilities error: %v\n", err)
		return
	}

	fmt.Printf("  RuntimeKind: %s\n", caps.RuntimeKind)
	fmt.Println("  Supported features:")
	for cap, support := range caps.Features {
		if support.Supported {
			fmt.Printf("    %s: yes (%s)\n", cap, support.Detail)
		}
	}
	if len(caps.Unsupported) > 0 {
		fmt.Printf("  Unsupported: %d features\n", len(caps.Unsupported))
	}
}

func demoRunFlow() {
	fmt.Println("\n=== FULL RUN FLOW DEMO ===")
	fmt.Println("Demonstrating a complete run flow with all wrappers stacked")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfgPath := os.Getenv("OPENCODE_CONFIG")
	if cfgPath == "" {
		cfgPath = "/home/antonioborgerees/coding/ultraplan/opencode-config.json"
	}

	rt := opencode.NewRuntime(
		opencode.WithEnv("OPENCODE_CONFIG="+cfgPath),
	)

	validating := agentwrap.ValidatingRuntime{
		Runtime: rt,
		Spec: agentwrap.ValidationSpec{
			Expectations: []agentwrap.ValidationExpectation{
				{
					ID:       "output-dir",
					Kind:     agentwrap.ExpectationDirectory,
					Path:     "reports/source",
					Severity: agentwrap.ExpectationRequired,
				},
			},
		},
	}

	backoff := agentwrap.FixedBackoff{DelayValue: 5 * time.Second}
	runner := agentwrap.PolicyRunner{
		Runtime: validating,
		Policy: agentwrap.BasicPolicy{
			MaxAttemptsPerTarget: 2,
			Backoff:              backoff,
			RetryRateLimits:       true,
		},
	}

	observing := agentwrap.ObservingRuntime{
		Runtime: runner,
		Store:   agentwrap.NewMemoryRunStore(),
	}

	req := agentwrap.RunRequest{
		Prompt:   "Write a simple hello world program in Go to /tmp/hello.go",
		WorkDir:  "/tmp",
		Provider: agentwrap.ProviderID("opencode"),
		Model:    agentwrap.ModelID("minimax-coding-plan/MiniMax-M2.7"),
		Timeout:  60 * time.Second,
		PermissionPolicy: &agentwrap.PermissionPolicy{
			Default: agentwrap.PermissionActionDeny,
			Tools: map[agentwrap.PermissionTool]agentwrap.PermissionAction{
				agentwrap.PermissionToolEdit:  agentwrap.PermissionActionAllow,
				agentwrap.PermissionToolShell: agentwrap.PermissionActionAllow,
				agentwrap.PermissionToolRead:  agentwrap.PermissionActionAllow,
			},
		},
		RequireHealth: []agentwrap.HealthCheckID{
			agentwrap.HealthCheckRuntimeAvailable,
		},
	}

	run, err := observing.StartRun(ctx, req)
	if err != nil {
		fmt.Printf("  StartRun error: %v\n", err)
		return
	}

	fmt.Printf("  Run started: ID=%s\n", run.ID())

	eventCount := 0
done:
	for {
		select {
		case ev, ok := <-run.Events():
			if !ok {
				break done
			}
			eventCount++
			if eventCount <= 3 {
				fmt.Printf("  Event %d: %s (%s)\n", eventCount, ev.Type, ev.Kind())
			}
		case <-ctx.Done():
			break done
		}
	}
	if eventCount > 3 {
		fmt.Printf("  ... and %d more events\n", eventCount-3)
	}

	result, err := run.Wait(ctx)
	if err != nil {
		fmt.Printf("  Wait error: %v\n", err)
	} else {
		fmt.Printf("  Result: Status=%s, Duration=%v\n", result.Status, result.Metadata.Duration)
		if result.Usage.InputTokens != nil {
			fmt.Printf("  Usage: Input=%d, Output=%d\n",
				*result.Usage.InputTokens, *result.Usage.OutputTokens)
		}
	}
}

func main() {
	demoValidation()
	demoPolicy()
	demoObservability()
	demoHealthChecks()
	demoPermissions()
	demoCapabilities()
	demoRunFlow()
}