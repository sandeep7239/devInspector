package remotepr

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Checkout struct {
	Path    string
	Cleanup func()
}

func Fetch(repo string, pr int) (Checkout, error) {
	if pr <= 0 {
		return Checkout{}, fmt.Errorf("pull request number must be greater than zero")
	}
	owner, name, err := parseRepo(repo)
	if err != nil {
		return Checkout{}, err
	}

	workspace, err := os.MkdirTemp("", "devinspector-pr-*")
	if err != nil {
		return Checkout{}, fmt.Errorf("create temporary workspace: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(workspace) }

	archivePath := filepath.Join(workspace, "pr.zip")
	url := fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/pull/%d/head", owner, name, pr)
	if err := download(url, archivePath); err != nil {
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

func parseRepo(repo string) (string, string, error) {
	repo = strings.TrimSpace(repo)
	repo = strings.TrimSuffix(repo, ".git")
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be owner/name or a GitHub repository URL")
	}
	return parts[0], parts[1], nil
}

func download(url, destination string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download PR archive: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download PR archive: GitHub returned %s", resp.Status)
	}

	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("save archive file: %w", err)
	}
	return nil
}

func unzip(source, destination string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return fmt.Errorf("open PR archive: %w", err)
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
	return "", fmt.Errorf("PR archive did not contain a repository folder")
}
