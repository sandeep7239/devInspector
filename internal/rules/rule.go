package rules

import (
	"path/filepath"
	"strings"

	"github.com/sandeep7239/devInspector/pkg/models"
)

type Rule interface {
	Name() string
	Description() string
	Match(file string) bool
	Check(file string, content string) []models.Issue
}

func BuiltIns() []Rule {
	return []Rule{
		DockerfileRule{},
		EnvSecurityRule{},
		DependencyVersionRule{},
	}
}

func EnabledRules(disabled []string) []Rule {
	blocked := map[string]bool{}
	for _, name := range disabled {
		blocked[strings.ToLower(strings.TrimSpace(name))] = true
	}

	var enabled []Rule
	for _, rule := range BuiltIns() {
		if !blocked[strings.ToLower(rule.Name())] {
			enabled = append(enabled, rule)
		}
	}
	return enabled
}

func baseName(path string) string {
	return strings.ToLower(filepath.Base(path))
}

func issue(file string, line int, severity models.Severity, rule, message, suggestion string) models.Issue {
	return models.Issue{
		File:       file,
		Line:       line,
		Severity:   severity,
		Rule:       rule,
		Message:    message,
		Suggestion: suggestion,
	}
}

func isSecretKey(key string) bool {
	lower := strings.ToLower(key)
	for _, marker := range []string{"password", "secret", "token", "api_key", "apikey", "private_key", "access_key"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
