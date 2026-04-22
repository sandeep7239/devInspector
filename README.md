# DevInspector

DevInspector is a production-readiness scanner for engineering projects. It checks Dockerfiles, environment files, and dependency manifests for risks that commonly block secure deployments: mutable image tags, leaked secrets, debug settings, and weak dependency pinning.

The project is built as a clean Go CLI with a pluggable rule engine, concurrent worker-pool scanning, structured output, Docker support, GitHub Actions integration, and a small REST API wrapper.

## Features

- Cobra-based CLI with `scan`, `version`, `config`, and `serve` commands
- Pluggable rule interface for adding new checks
- Built-in Dockerfile validation, `.env` security checks, and dependency/version checks
- Concurrent file scanning with a worker pool
- Table and JSON output modes
- YAML configuration through `.devinspector.yaml`
- Multi-stage Docker image
- GitHub Actions workflow that builds and scans on pull requests
- REST API endpoint for scan automation

## Installation

```bash
go build -o devinspector ./cmd/app
```

Run the binary:

```bash
./devinspector version
```

## Usage

Scan the current directory:

```bash
./devinspector scan .
```

Return structured JSON:

```bash
./devinspector scan --format=json .
```

Create a config file:

```bash
./devinspector config
```

Start the REST API wrapper:

```bash
./devinspector serve --port=8080
```

Scan through the API:

```bash
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d '{"path":"."}'
```

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

## CI

The repository includes `.github/workflows/scan.yml`. It builds `devinspector` and runs:

```bash
./devinspector scan --format=json .
```

Critical findings return a non-zero exit code when `failOnCritical` is enabled.
