package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/sandeep7239/devInspector/internal/rules"
	"github.com/sandeep7239/devInspector/internal/scanner"
	"github.com/sandeep7239/devInspector/internal/server"
	"github.com/sandeep7239/devInspector/internal/utils"
	"github.com/sandeep7239/devInspector/pkg/models"
	"github.com/spf13/cobra"
)

const Version = "1.1.0"

var (
	outputFormat string
	logLevel     string
	port         string
	prRepo       string
	prNumber     int
)

func Execute() {
	root := &cobra.Command{
		Use:   "devinspector",
		Short: "Production readiness scanner for Docker, environment, and dependency hygiene",
		Long:  "DevInspector scans repositories and pull requests for container, secret, and dependency risks using a pluggable rule engine.",
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "WARN", "Log level: INFO, WARN, ERROR")
	root.AddCommand(scanCommand(), scanPRCommand(), versionCommand(), configCommand(), serveCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a local project directory",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			if err := runScan(path, outputFormat); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format: table or json")
	return cmd
}

func scanPRCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan-pr --repo <github-repo-url-or-owner/name> --pr <number>",
		Short: "Fetch and scan a remote GitHub pull request",
		Run: func(cmd *cobra.Command, args []string) {
			if prRepo == "" || prNumber <= 0 {
				fmt.Fprintln(os.Stderr, "scan-pr requires --repo and --pr")
				os.Exit(1)
			}
			if err := runRemotePRScan(prRepo, prNumber, outputFormat); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVar(&prRepo, "repo", "", "GitHub repository URL or owner/name, for example https://github.com/org/repo or org/repo")
	cmd.Flags().IntVar(&prNumber, "pr", 0, "Pull request number to scan")
	cmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format: table or json")
	return cmd
}

func runRemotePRScan(repo string, pr int, format string) error {
	repoURL := normalizeRepoURL(repo)
	tmpDir, err := os.MkdirTemp("", "devinspector-pr-*")
	if err != nil {
		return fmt.Errorf("create temporary checkout: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := runGit("clone", "--quiet", "--depth", "1", repoURL, tmpDir); err != nil {
		return fmt.Errorf("clone repository: %w", err)
	}

	branch := fmt.Sprintf("devinspector-pr-%d", pr)
	refspec := fmt.Sprintf("pull/%d/head:%s", pr, branch)
	if err := runGitIn(tmpDir, "fetch", "--quiet", "origin", refspec); err != nil {
		return fmt.Errorf("fetch pull request #%d: %w", pr, err)
	}
	if err := runGitIn(tmpDir, "checkout", "--quiet", branch); err != nil {
		return fmt.Errorf("checkout pull request #%d: %w", pr, err)
	}

	fmt.Fprintf(os.Stderr, "Scanning PR #%d from %s\n", pr, repoURL)
	return runScan(tmpDir, format)
}

func normalizeRepoURL(repo string) string {
	repo = strings.TrimSpace(repo)
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "git@") {
		return repo
	}
	return "https://github.com/" + strings.TrimSuffix(repo, ".git") + ".git"
}

func runGit(args ...string) error {
	return runGitIn("", args...)
}

func runGitIn(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("git %s failed: %s", strings.Join(args, " "), message)
	}
	return nil
}

func runScan(projectPath string, format string) error {
	logger := utils.NewLogger(logLevel)
	if _, err := os.Stat(projectPath); err != nil {
		return fmt.Errorf("scan path is not accessible: %w", err)
	}

	cfg, err := utils.LoadConfig(projectPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger.Info("starting scan for %s", projectPath)
	engine := scanner.New(rules.EnabledRules(cfg.DisabledRules), cfg.WorkerCount)
	result, err := engine.Scan(projectPath)
	if err != nil {
		return err
	}

	switch format {
	case "json":
		if err := printJSON(result); err != nil {
			return err
		}
	case "table", "":
		printTable(result)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}

	if cfg.FailOnCritical && result.CriticalIssues > 0 {
		return fmt.Errorf("scan found %d critical issue(s)", result.CriticalIssues)
	}
	return nil
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("DevInspector %s\n", Version)
		},
	}
}

func configCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Create a default .devinspector.yaml",
		Run: func(cmd *cobra.Command, args []string) {
			if err := utils.WriteDefaultConfig("."); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("Created .devinspector.yaml")
		},
	}
}

func serveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the REST API wrapper",
		Run: func(cmd *cobra.Command, args []string) {
			if err := server.Start(port, logLevel); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVar(&port, "port", "8080", "HTTP server port")
	return cmd
}

func printJSON(result models.ScanResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func printTable(result models.ScanResult) {
	header := color.New(color.FgCyan, color.Bold)
	ok := color.New(color.FgGreen, color.Bold)
	warn := color.New(color.FgYellow, color.Bold)
	fail := color.New(color.FgRed, color.Bold)

	header.Println("DevInspector Scan Report")
	fmt.Printf("Project: %s\n", result.ProjectPath)
	fmt.Printf("Score:   %d/100\n", result.OverallScore)
	fmt.Printf("Issues:  %d total, %d critical\n\n", result.TotalIssues, result.CriticalIssues)

	if len(result.Results) == 0 {
		ok.Println("No matching files found.")
		return
	}

	fmt.Printf("%-10s %-24s %-6s %s\n", "Severity", "Rule", "Line", "File")
	fmt.Println("---------- ------------------------ ------ ----")
	for _, file := range result.Results {
		if len(file.Issues) == 0 {
			ok.Printf("%-10s %-24s %-6s %s\n", "OK", file.FileType, "-", file.FilePath)
			continue
		}
		for _, issue := range file.Issues {
			printIssueLine(issue, warn, fail)
		}
	}
}

func printIssueLine(issue models.Issue, warn *color.Color, fail *color.Color) {
	line := "-"
	if issue.Line > 0 {
		line = fmt.Sprintf("%d", issue.Line)
	}
	row := fmt.Sprintf("%-10s %-24s %-6s %s\n", issue.Severity, issue.Rule, line, issue.File)
	switch issue.Severity {
	case models.SeverityCritical, models.SeverityError:
		fail.Print(row)
	case models.SeverityWarning:
		warn.Print(row)
	default:
		fmt.Print(row)
	}
	fmt.Printf("  %s\n", issue.Message)
	if issue.Suggestion != "" {
		fmt.Printf("  Suggestion: %s\n", issue.Suggestion)
	}
}
