package remotepr

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxArchiveBytes int64 = 80 << 20

var httpClient = &http.Client{Timeout: 30 * time.Second}

type Checkout struct {
	Path    string
	Cleanup func()
}

type repositoryInfo struct {
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
}

type pullRequestInfo struct {
	Number int    `json:"number"`
	State  string `json:"state"`
	Title  string `json:"title"`
}

type githubError struct {
	Message string `json:"message"`
}

func Fetch(repo string, pr int) (Checkout, error) {
	if pr <= 0 {
		return Checkout{}, fmt.Errorf("enter a valid pull request number, or use repository scan when no PR exists")
	}
	owner, name, err := parseRepo(repo)
	if err != nil {
		return Checkout{}, err
	}
	if _, err := getRepository(owner, name); err != nil {
		return Checkout{}, err
	}
	if _, err := getPullRequest(owner, name, pr); err != nil {
		return Checkout{}, err
	}

	url := fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/pull/%d/head", owner, name, pr)
	return fetchArchive(url, "pr")
}

func FetchRepository(repo string) (Checkout, error) {
	owner, name, err := parseRepo(repo)
	if err != nil {
		return Checkout{}, err
	}
	info, err := getRepository(owner, name)
	if err != nil {
		return Checkout{}, err
	}
	if info.DefaultBranch == "" {
		return Checkout{}, fmt.Errorf("repository default branch could not be detected")
	}

	url := fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/heads/%s", owner, name, info.DefaultBranch)
	return fetchArchive(url, "repo")
}

func parseRepo(repo string) (string, string, error) {
	repo = strings.TrimSpace(repo)
	repo = strings.TrimSuffix(repo, "/")
	repo = strings.TrimSuffix(repo, ".git")
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	if strings.HasPrefix(repo, "git@github.com:") {
		repo = strings.TrimPrefix(repo, "git@github.com:")
	}
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be owner/name or a GitHub repository URL")
	}
	return parts[0], parts[1], nil
}

func getRepository(owner, name string) (repositoryInfo, error) {
	var info repositoryInfo
	status, err := getGitHubJSON(fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, name), &info)
	if err != nil {
		return info, err
	}
	switch status {
	case http.StatusOK:
		return info, nil
	case http.StatusNotFound:
		return info, fmt.Errorf("repository %s/%s was not found or is private; public scans need a public GitHub repo", owner, name)
	case http.StatusForbidden:
		return info, fmt.Errorf("GitHub API rate limit or access restriction reached; try again later or configure a GITHUB_TOKEN")
	default:
		return info, fmt.Errorf("GitHub repository lookup failed with status %d", status)
	}
}

func getPullRequest(owner, name string, pr int) (pullRequestInfo, error) {
	var info pullRequestInfo
	status, err := getGitHubJSON(fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", owner, name, pr), &info)
	if err != nil {
		return info, err
	}
	switch status {
	case http.StatusOK:
		return info, nil
	case http.StatusNotFound:
		return info, fmt.Errorf("pull request #%d was not found in %s/%s; scan the repository instead if no PR is open", pr, owner, name)
	case http.StatusForbidden:
		return info, fmt.Errorf("GitHub API rate limit or access restriction reached; try again later or configure a GITHUB_TOKEN")
	default:
		return info, fmt.Errorf("GitHub pull request lookup failed with status %d", status)
	}
}

func getGitHubJSON(url string, target any) (int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DevInspector")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("connect to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var ghErr githubError
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&ghErr)
		if ghErr.Message != "" && resp.StatusCode != http.StatusNotFound {
			return resp.StatusCode, fmt.Errorf("GitHub API error: %s", ghErr.Message)
		}
		return resp.StatusCode, nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(target); err != nil {
		return resp.StatusCode, fmt.Errorf("decode GitHub response: %w", err)
	}
	return resp.StatusCode, nil
}

func fetchArchive(url, label string) (Checkout, error) {
	workspace, err := os.MkdirTemp("", "devinspector-"+label+"-*")
	if err != nil {
		return Checkout{}, fmt.Errorf("create temporary workspace: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(workspace) }

	archivePath := filepath.Join(workspace, label+".zip")
	if err := download(url, archivePath, label); err != nil {
		cleanup()
		return Checkout{}, err
	}

	extractPath := filepath.Join(workspace, "src")
	if err := unzip(archivePath, extractPath); err != nil {
		cleanup()
		return Checkout{}, err
	}

	root, err := firstChildDir(extractPath)
	if err != nil {
		cleanup()
		return Checkout{}, err
	}

	return Checkout{Path: root, Cleanup: cleanup}, nil
}

func download(url, destination, label string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "DevInspector")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s archive: %w", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s archive failed with %s", label, resp.Status)
	}
	if resp.ContentLength > maxArchiveBytes {
		return fmt.Errorf("repository archive is too large for hosted scanning; try the CLI locally")
	}

	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	defer file.Close()

	limited := &io.LimitedReader{R: resp.Body, N: maxArchiveBytes + 1}
	if _, err := io.Copy(file, limited); err != nil {
		return fmt.Errorf("save archive file: %w", err)
	}
	if limited.N <= 0 {
		return fmt.Errorf("repository archive is too large for hosted scanning; try the CLI locally")
	}
	return nil
}

func unzip(source, destination string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer reader.Close()

	cleanDestination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(cleanDestination, file.Name)
		if !strings.HasPrefix(path, cleanDestination+string(os.PathSeparator)) && path != cleanDestination {
			return fmt.Errorf("archive contains unsafe path %q", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		_ = in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func firstChildDir(parent string) (string, error) {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(parent, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("archive did not contain a repository folder")
}
