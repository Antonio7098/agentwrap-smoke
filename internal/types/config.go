package types

type Config struct {
	DefaultModel               string
	PrimaryModel               string
	BackupModel                string
	DefaultVariant            string
	DefaultParallel            int
	DefaultTimeoutMs           int64
	SprintPlanningModel        string
	SprintPlanningContextWindow int64
	SprintExecutionModel       string
	SprintExecutionVariant     string
}

var DefaultConfig = Config{
	DefaultModel:                "minimax-coding-plan/MiniMax-M2.7",
	PrimaryModel:                "minimax-coding-plan/MiniMax-M2.7",
	BackupModel:                "opencode/deepseek-v4-flash-free",
	DefaultVariant:             "high",
	DefaultParallel:            3,
	DefaultTimeoutMs:           1_800_000,
	SprintPlanningModel:        "openai/gpt-5.5",
	SprintPlanningContextWindow: 1_000_000,
	SprintExecutionModel:       "openai/gpt-5.5",
	SprintExecutionVariant:     "low",
}