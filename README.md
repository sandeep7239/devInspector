# DevInspector

DevInspector is a production-readiness scanner for engineering projects. It checks Dockerfiles, environment files, and dependency manifests for risks that commonly block secure deployments: mutable image tags, leaked secrets, debug settings, and weak dependency pinning.

The project includes a Go CLI, a built-in browser dashboard, a REST API, a pluggable rule engine, concurrent worker-pool scanning, Docker support, and GitHub Actions PR validation.

## What It Checks

DevInspector does not judge the full business logic of an application. It checks high-signal DevOps and deployment risks that are common in pull requests and repositories.

Current checks include:

- Dockerfile validation: latest tags, unpinned images, root containers, missing health checks, broad `COPY . .`, missing `.dockerignore`, and build-time secrets.
- Environment security: secret-like values in `.env` files and risky production defaults like `DEBUG=true`.
- Dependency/version hygiene: weak or floating versions in `go.mod`, `package.json`, and `requirements.txt`.

A repository or PR is considered risky when critical issues are found. In CI, `failOnCritical: true` makes the workflow fail on critical findings.

## Features

- Cobra-based CLI with `scan`, `scan-pr`, `version`, `config`, and `serve` commands
- Built-in browser dashboard at `http://localhost:8080`
- REST API endpoint for scan automation
- Pluggable rule interface for adding new checks
- Concurrent file scanning with a worker pool
- Table and JSON output modes
- YAML configuration through `.devinspector.yaml`
- Multi-stage Docker image
- GitHub Actions workflow that builds and scans on pull requests

## Installation

```bash
go build -o devinspector ./cmd/app
```

On Windows:

```powershell
go build -o devinspector.exe .\cmd\app
```

## Run The Dashboard

```powershell
cd C:\Users\Sandeep\devInspector
.\devinspector.exe serve --port=8080
```

Open:

```text
http://localhost:8080
```

Use `.` to scan the current repository, or paste another local repository path.

## CLI Usage

```powershell
.\devinspector.exe version
.\devinspector.exe scan .
.\devinspector.exe scan --format=json .
.\devinspector.exe scan-pr --repo sandeep7239/devInspector --pr 1
.\devinspector.exe config
```

## API Usage

Start the server:

```powershell
.\devinspector.exe serve --port=8080
```

Check health:

```powershell
curl http://localhost:8080/health
```

Run a scan:

```powershell
curl -X POST http://localhost:8080/scan -H "Content-Type: application/json" -d "{\"path\":\".\"}"
```

## Validate A Remote GitHub PR

Use `scan-pr` when the pull request exists on GitHub and you do not already have the branch locally. DevInspector creates a temporary checkout, fetches the PR branch, scans it, prints the result, and deletes the temporary folder.

```powershell
.\devinspector.exe scan-pr --repo owner/repo --pr 12
```

You can also pass a full repo URL:

```powershell
.\devinspector.exe scan-pr --repo https://github.com/owner/repo --pr 12 --format=json
```

For private repositories, your local `git` must already be authenticated through GitHub CLI, SSH, or Git Credential Manager.

## Validate Another PR Manually

Clone the target repository and checkout the pull request branch:

```powershell
git clone <repo-url>
cd <repo-folder>
git checkout <pr-branch>
```

Run DevInspector against that checkout:

```powershell
C:\Users\Sandeep\devInspector\devinspector.exe scan .
```

Or use the dashboard and enter the full path of that checked-out repository.

## Validate PRs Automatically

Add DevInspector's GitHub Actions workflow to the repository you want to protect. On every pull request, GitHub Actions builds the scanner and runs:

```bash
./devinspector scan --format=json .
```

If critical issues are found and `failOnCritical` is enabled, the PR check fails.

## Configuration

`.devinspector.yaml`

```yaml
disabledRules: []
workerCount: 5
failOnCritical: true
```

Disable a rule by name:

```yaml
disabledRules:
  - env-security
workerCount: 8
failOnCritical: true
```

Built-in rules:

- `dockerfile-validation`
- `env-security`
- `dependency-version`

## Docker

Build the image:

```bash
docker build -t devinspector .
```

Run the scanner inside the image:

```bash
docker run --rm -v "$PWD:/workspace" devinspector scan /workspace
```

## Architecture

```text
/cmd
  /app
    main.go
/internal
  /analyzer
  /rules
  /scanner
  /server
  /utils
/pkg
  /models
```

Core rule contract:

```go
type Rule interface {
    Name() string
    Description() string
    Match(file string) bool
    Check(file string, content string) []models.Issue
}
```

## Testing

```powershell
go test ./...
go build -o devinspector.exe .\cmd\app
.\devinspector.exe scan --format=json .
```