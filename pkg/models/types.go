package models

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityError    Severity = "ERROR"
	SeverityWarning  Severity = "WARNING"
	SeverityInfo     Severity = "INFO"
)

type Issue struct {
	File       string   `json:"file" yaml:"file"`
	Line       int      `json:"line" yaml:"line"`
	Severity   Severity `json:"severity" yaml:"severity"`
	Rule       string   `json:"rule" yaml:"rule"`
	Message    string   `json:"message" yaml:"message"`
	Suggestion string   `json:"suggestion,omitempty" yaml:"suggestion,omitempty"`
}

type FileResult struct {
	FilePath string  `json:"filePath" yaml:"filePath"`
	FileType string  `json:"fileType" yaml:"fileType"`
	Issues   []Issue `json:"issues" yaml:"issues"`
	Score    int     `json:"score" yaml:"score"`
}

type ScanResult struct {
	ProjectPath    string       `json:"projectPath" yaml:"projectPath"`
	Results        []FileResult `json:"results" yaml:"results"`
	OverallScore   int          `json:"overallScore" yaml:"overallScore"`
	TotalIssues    int          `json:"totalIssues" yaml:"totalIssues"`
	CriticalIssues int          `json:"criticalIssues" yaml:"criticalIssues"`
}

type Config struct {
	DisabledRules  []string `json:"disabledRules" yaml:"disabledRules"`
	WorkerCount    int      `json:"workerCount" yaml:"workerCount"`
	FailOnCritical bool     `json:"failOnCritical" yaml:"failOnCritical"`
}
