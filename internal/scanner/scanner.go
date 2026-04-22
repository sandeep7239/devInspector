package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/sandeep/devinspector/internal/rules"
	"github.com/sandeep/devinspector/pkg/models"
)

type Scanner struct {
	rules       []rules.Rule
	workerCount int
}

func New(ruleSet []rules.Rule, workerCount int) *Scanner {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}
	return &Scanner{rules: ruleSet, workerCount: workerCount}
}

func (s *Scanner) Scan(projectPath string) (models.ScanResult, error) {
	files, err := s.discover(projectPath)
	if err != nil {
		return models.ScanResult{}, err
	}

	jobs := make(chan string)
	results := make(chan models.FileResult)
	var wg sync.WaitGroup

	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				if result, ok := s.scanFile(file); ok {
					results <- result
				}
			}
		}()
	}

	go func() {
		for _, file := range files {
			jobs <- file
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var fileResults []models.FileResult
	for result := range results {
		fileResults = append(fileResults, result)
	}
	sort.Slice(fileResults, func(i, j int) bool {
		return fileResults[i].FilePath < fileResults[j].FilePath
	})

	return summarize(projectPath, fileResults), nil
}

func (s *Scanner) discover(projectPath string) ([]string, error) {
	var files []string
	root, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if shouldSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		for _, rule := range s.rules {
			if rule.Match(path) {
				files = append(files, path)
				break
			}
		}
		return nil
	})
	return files, err
}

func (s *Scanner) scanFile(file string) (models.FileResult, bool) {
	data, err := os.ReadFile(file)
	if err != nil {
		return models.FileResult{}, false
	}

	issues := []models.Issue{}
	fileType := "matched"
	for _, rule := range s.rules {
		if !rule.Match(file) {
			continue
		}
		fileType = rule.Name()
		issues = append(issues, rule.Check(file, string(data))...)
	}

	return models.FileResult{
		FilePath: file,
		FileType: fileType,
		Issues:   issues,
		Score:    score(issues),
	}, true
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".idea", ".vscode", "node_modules", "vendor", "dist", "build", "target", "tmp":
		return true
	default:
		return false
	}
}

func summarize(projectPath string, results []models.FileResult) models.ScanResult {
	out := models.ScanResult{ProjectPath: projectPath, Results: results}
	totalScore := 0
	for _, result := range results {
		totalScore += result.Score
		for _, issue := range result.Issues {
			out.TotalIssues++
			if issue.Severity == models.SeverityCritical {
				out.CriticalIssues++
			}
		}
	}
	if len(results) == 0 {
		out.OverallScore = 100
	} else {
		out.OverallScore = totalScore / len(results)
	}
	return out
}

func score(issues []models.Issue) int {
	score := 100
	for _, issue := range issues {
		switch issue.Severity {
		case models.SeverityCritical:
			score -= 25
		case models.SeverityError:
			score -= 15
		case models.SeverityWarning:
			score -= 10
		default:
			score -= 5
		}
	}
	if score < 0 {
		return 0
	}
	return score
}
