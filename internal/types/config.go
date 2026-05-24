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
	DefaultModel:                "opencode/deepseek-v4-flash-free",
	PrimaryModel:                "opencode/deepseek-v4-flash-free",
	BackupModel:                 "opencode/deepseek-v4-flash-free",
	DefaultVariant:              "high",
	DefaultParallel:            3,
	DefaultTimeoutMs:           1_800_000,
	SprintPlanningModel:        "opencode/deepseek-v4-flash-free",
	SprintPlanningContextWindow: 1_000_000,
	SprintExecutionModel:       "opencode/deepseek-v4-flash-free",
	SprintExecutionVariant:     "low",
}