package rules

import (
	"strings"

	"github.com/sandeep7239/devInspector/pkg/models"
)

type EnvSecurityRule struct{}

func (EnvSecurityRule) Name() string        { return "env-security" }
func (EnvSecurityRule) Description() string { return "Detects risky values in .env files." }

func (EnvSecurityRule) Match(file string) bool {
	name := baseName(file)
	return name == ".env" || strings.HasPrefix(name, ".env.")
}

func (EnvSecurityRule) Check(file string, content string) []models.Issue {
	var issues []models.Issue
	for idx, raw := range strings.Split(content, "\n") {
		lineNo := idx + 1
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		if isSecretKey(key) && value != "" && !strings.Contains(value, "changeme") && !strings.Contains(value, "example") {
			issues = append(issues, issue(file, lineNo, models.SeverityCritical, "ENV_HARDCODED_SECRET", "A secret-like key has a concrete value in an env file.", "Move real secrets to a vault or deployment secret store and keep only examples in source control."))
		}
		if strings.EqualFold(key, "DEBUG") && strings.EqualFold(value, "true") {
			issues = append(issues, issue(file, lineNo, models.SeverityWarning, "ENV_DEBUG_ENABLED", "DEBUG=true should not be used in production.", "Use environment-specific config and default production deployments to DEBUG=false."))
		}
	}
	return issues
}
