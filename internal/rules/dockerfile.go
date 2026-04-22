package rules

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sandeep7239/devInspector/pkg/models"
)

type DockerfileRule struct{}

func (DockerfileRule) Name() string { return "dockerfile-validation" }
func (DockerfileRule) Description() string {
	return "Validates Dockerfiles for reproducibility and container security."
}

func (DockerfileRule) Match(file string) bool {
	name := baseName(file)
	return name == "dockerfile" ||
		(strings.HasPrefix(name, "dockerfile.") && !strings.HasSuffix(name, ".go"))
}

func (DockerfileRule) Check(file string, content string) []models.Issue {
	var issues []models.Issue
	lines := strings.Split(content, "\n")
	hasUser := false
	hasHealthcheck := false

	for idx, raw := range lines {
		lineNo := idx + 1
		line := strings.TrimSpace(raw)
		upper := strings.ToUpper(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(upper, "FROM ") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				image := fields[1]
				switch {
				case strings.Contains(strings.ToLower(image), ":latest"):
					issues = append(issues, issue(file, lineNo, models.SeverityCritical, "DOCKER_LATEST_TAG", "Base image uses the mutable latest tag.", "Pin the image to a specific version or digest."))
				case !strings.Contains(image, ":") && !strings.Contains(image, "@sha256:"):
					issues = append(issues, issue(file, lineNo, models.SeverityWarning, "DOCKER_UNPINNED_IMAGE", "Base image has no explicit version.", "Use a stable semantic version or sha256 digest."))
				case hasFloatingDistroTag(image):
					issues = append(issues, issue(file, lineNo, models.SeverityInfo, "DOCKER_PARTIAL_PIN", "Base image is pinned only to a moving distro tag.", "Use a full version tag or immutable digest for stronger reproducibility."))
				}
			}
		}

		if strings.HasPrefix(upper, "ENV ") || strings.HasPrefix(upper, "ARG ") {
			if isSecretKey(line) {
				issues = append(issues, issue(file, lineNo, models.SeverityCritical, "DOCKER_SECRET_IN_BUILD", "Build file appears to contain a secret-like ENV or ARG.", "Inject secrets at runtime through your orchestrator or secret manager."))
			}
		}

		if strings.HasPrefix(upper, "COPY . .") || strings.HasPrefix(upper, "COPY ./ .") {
			issues = append(issues, issue(file, lineNo, models.SeverityWarning, "DOCKER_COPY_ALL", "COPY . . can send credentials and build artifacts into the image.", "Copy only required files and maintain a strict .dockerignore."))
		}
		if strings.HasPrefix(upper, "USER ") {
			hasUser = true
		}
		if strings.HasPrefix(upper, "HEALTHCHECK ") {
			hasHealthcheck = true
		}
	}

	if !hasUser {
		issues = append(issues, issue(file, 0, models.SeverityWarning, "DOCKER_RUNS_AS_ROOT", "No USER instruction found.", "Create and switch to a non-root user in the final image stage."))
	}
	if !hasHealthcheck {
		issues = append(issues, issue(file, 0, models.SeverityInfo, "DOCKER_NO_HEALTHCHECK", "No HEALTHCHECK instruction found.", "Add a lightweight health check for runtime observability."))
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(file), ".dockerignore")); err != nil {
		issues = append(issues, issue(file, 0, models.SeverityWarning, "DOCKER_NO_IGNORE_FILE", ".dockerignore is missing next to this Dockerfile.", "Create .dockerignore to keep secrets, VCS data, and build output out of the build context."))
	}
	return issues
}

func hasFloatingDistroTag(image string) bool {
	lower := strings.ToLower(image)
	return strings.HasSuffix(lower, ":alpine") ||
		strings.HasSuffix(lower, ":slim") ||
		strings.HasSuffix(lower, ":latest")
}
