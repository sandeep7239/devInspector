package rules

import (
	"strings"

	"github.com/sandeep/devinspector/pkg/models"
)

type DependencyVersionRule struct{}

func (DependencyVersionRule) Name() string { return "dependency-version" }
func (DependencyVersionRule) Description() string {
	return "Finds weak or floating dependency version declarations."
}

func (DependencyVersionRule) Match(file string) bool {
	name := baseName(file)
	return name == "go.mod" || name == "package.json" || name == "requirements.txt"
}

func (DependencyVersionRule) Check(file string, content string) []models.Issue {
	switch baseName(file) {
	case "go.mod":
		return checkGoMod(file, content)
	case "package.json":
		return checkPackageJSON(file, content)
	case "requirements.txt":
		return checkRequirements(file, content)
	default:
		return nil
	}
}

func checkGoMod(file, content string) []models.Issue {
	var issues []models.Issue
	for idx, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if strings.Contains(line, "=>") && strings.Contains(line, "../") {
			issues = append(issues, issue(file, idx+1, models.SeverityWarning, "GO_LOCAL_REPLACE", "go.mod uses a local replace directive.", "Avoid local replace directives in production modules."))
		}
		if strings.Contains(line, "-") && strings.Contains(line, ".") && strings.Contains(line, "v0.0.0-") {
			issues = append(issues, issue(file, idx+1, models.SeverityInfo, "GO_PSEUDO_VERSION", "Dependency uses a pseudo-version.", "Prefer tagged releases for easier auditing and upgrades."))
		}
	}
	return issues
}

func checkPackageJSON(file, content string) []models.Issue {
	var issues []models.Issue
	for idx, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if strings.Contains(line, `": "*"`) || strings.Contains(line, `": "latest"`) {
			issues = append(issues, issue(file, idx+1, models.SeverityCritical, "NPM_FLOATING_VERSION", "package.json contains a floating dependency version.", "Pin dependencies with lockfiles and reviewed version ranges."))
		}
	}
	return issues
}

func checkRequirements(file, content string) []models.Issue {
	var issues []models.Issue
	for idx, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "==") {
			issues = append(issues, issue(file, idx+1, models.SeverityWarning, "PYTHON_UNPINNED_DEPENDENCY", "Python dependency is not pinned with ==.", "Pin Python dependencies for reproducible builds."))
		}
	}
	return issues
}
