package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sandeep7239/devInspector/internal/remotepr"
	"github.com/sandeep7239/devInspector/internal/rules"
	"github.com/sandeep7239/devInspector/internal/scanner"
	"github.com/sandeep7239/devInspector/internal/utils"
	"github.com/sandeep7239/devInspector/pkg/models"
)

type scanRequest struct {
	Path string `json:"path"`
}

type remoteRepoRequest struct {
	Repo string `json:"repo"`
	PR   int    `json:"pr"`
}

func Handler() http.Handler {
	return newMux()
}

func Start(port string, logLevel string) error {
	logger := utils.NewLogger(logLevel)
	addr := ":" + port
	logger.Info("DevInspector UI and API listening on http://localhost%s", addr)
	return http.ListenAndServe(addr, newMux())
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", dashboardHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "DevInspector"})
	})
	mux.HandleFunc("/scan", scanHandler)
	mux.HandleFunc("/scan-repo", scanRepoHandler)
	mux.HandleFunc("/scan-pr", scanPRHandler)
	return mux
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Path == "" {
		req.Path = "."
	}

	cfg, err := utils.LoadConfig(req.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result, err := scanner.New(rules.EnabledRules(cfg.DisabledRules), cfg.WorkerCount).Scan(req.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func scanRepoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var req remoteRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Repo == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "GitHub repository is required"})
		return
	}

	checkout, err := remotepr.FetchRepository(req.Repo)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	defer checkout.Cleanup()

	result, err := scanCheckout(checkout.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func scanPRHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var req remoteRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Repo == "" || req.PR <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "GitHub repository and PR number are required"})
		return
	}

	checkout, err := remotepr.Fetch(req.Repo, req.PR)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	defer checkout.Cleanup()

	result, err := scanCheckout(checkout.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func scanCheckout(path string) (models.ScanResult, error) {
	cfg := utils.DefaultConfig()
	return scanner.New(rules.EnabledRules(cfg.DisabledRules), cfg.WorkerCount).Scan(path)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(dashboardHTML))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		fmt.Fprintf(w, `{"error":%q}`, err.Error())
	}
}

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>DevInspector</title>
  <style>
    :root { font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color: #102033; background: #eef3f8; }
    * { box-sizing: border-box; }
    body { margin: 0; background: #eef3f8; color: #102033; }
    .app { min-height: 100vh; }
    .topbar { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 18px min(5vw, 56px); background: #0b1220; color: white; border-bottom: 1px solid #1e293b; }
    .brand { display: flex; align-items: center; gap: 12px; font-size: 20px; font-weight: 800; }
    .mark { width: 36px; height: 36px; display: grid; place-items: center; border-radius: 8px; background: #22c55e; color: #052e16; font-weight: 900; }
    .owner { color: #aab7c7; font-size: 14px; }
    main { width: min(1180px, calc(100% - 32px)); margin: 22px auto 44px; display: grid; gap: 18px; }
    .hero { display: grid; grid-template-columns: minmax(0, 1.1fr) 340px; gap: 18px; align-items: stretch; }
    .panel, .card { background: #ffffff; border: 1px solid #d7e0ea; border-radius: 8px; box-shadow: 0 10px 26px rgba(15, 23, 42, 0.07); }
    .panel { padding: 22px; }
    h1, h2, h3, p { margin-top: 0; }
    h1 { font-size: 38px; line-height: 1.08; letter-spacing: 0; margin-bottom: 12px; color: #0f172a; }
    h2 { font-size: 22px; margin-bottom: 14px; color: #0f172a; }
    h3 { font-size: 16px; margin-bottom: 8px; color: #0f172a; }
    p, .muted { color: #526176; line-height: 1.58; }
    .mode-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
    .card { padding: 18px; }
    form { display: grid; gap: 12px; }
    label { font-weight: 800; color: #172033; }
    .row { display: grid; grid-template-columns: minmax(0, 1fr) 160px auto; gap: 10px; }
    .repo-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 10px; }
    input { width: 100%; border: 1px solid #c9d5e2; border-radius: 7px; padding: 12px 13px; font-size: 15px; background: white; }
    input:focus { outline: 3px solid #bfdbfe; border-color: #2563eb; }
    button { border: 0; border-radius: 7px; padding: 12px 16px; min-height: 46px; background: #2563eb; color: white; font-weight: 800; cursor: pointer; white-space: nowrap; }
    button.secondary { background: #0f766e; }
    button:disabled { opacity: .68; cursor: wait; }
    .hint { color: #66758a; font-size: 14px; margin: 0; }
    .summary { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
    .metric { background: #f8fafc; border: 1px solid #dce5ef; border-radius: 8px; padding: 14px; }
    .metric span { color: #66758a; font-size: 13px; font-weight: 800; }
    .metric strong { display: block; font-size: 28px; margin-top: 4px; color: #0f172a; }
    .toolbar { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
    .status { color: #526176; font-weight: 700; }
    table { width: 100%; border-collapse: collapse; font-size: 14px; }
    th, td { text-align: left; border-bottom: 1px solid #e2e8f0; padding: 12px 10px; vertical-align: top; }
    th { font-size: 12px; text-transform: uppercase; color: #475569; background: #f8fafc; }
    .badge { display: inline-block; border-radius: 999px; padding: 3px 9px; font-size: 12px; font-weight: 900; background: #e2e8f0; color: #334155; }
    .CRITICAL, .ERROR { background: #fee2e2; color: #991b1b; }
    .WARNING { background: #fef3c7; color: #92400e; }
    .INFO { background: #dbeafe; color: #1e40af; }
    .OK { background: #dcfce7; color: #166534; }
    .rules { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
    .json { white-space: pre-wrap; background: #0b1220; color: #dbeafe; border-radius: 8px; padding: 14px; max-height: 340px; overflow: auto; font-size: 13px; }
    code, pre { font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace; }
    @media (max-width: 900px) { .hero, .mode-grid, .rules, .row, .repo-row { grid-template-columns: 1fr; } .topbar { align-items: flex-start; flex-direction: column; } }
  </style>
</head>
<body>
  <div class="app">
    <header class="topbar">
      <div class="brand"><span class="mark">DI</span><span>DevInspector</span></div>
      <div class="owner">Production readiness checks by Sandeep</div>
    </header>

    <main>
      <section class="hero">
        <div class="panel">
          <h1>Scan repositories and pull requests before deployment.</h1>
          <p>DevInspector checks Dockerfiles, environment files, and dependency manifests for risky DevOps patterns. Hosted scans work with public GitHub repositories and pull requests.</p>
        </div>
        <aside class="summary" id="summary">
          <div class="metric"><span>Score</span><strong>-</strong></div>
          <div class="metric"><span>Total issues</span><strong>-</strong></div>
          <div class="metric"><span>Critical</span><strong>-</strong></div>
          <div class="metric"><span>Files scanned</span><strong>-</strong></div>
        </aside>
      </section>

      <section class="mode-grid">
        <div class="card">
          <h2>Remote Repository</h2>
          <form id="repo-form">
            <label for="repo-only">GitHub repository</label>
            <div class="repo-row"><input id="repo-only" placeholder="owner/repo or https://github.com/owner/repo"><button class="secondary" id="repo-button" type="submit">Scan Repo</button></div>
            <p class="hint">Use this when the repository has no pull request yet.</p>
          </form>
        </div>

        <div class="card">
          <h2>Remote Pull Request</h2>
          <form id="pr-form">
            <label for="repo">GitHub repository and PR</label>
            <div class="row"><input id="repo" placeholder="owner/repo"><input id="pr" type="number" min="1" placeholder="PR number"><button id="pr-button" type="submit">Scan PR</button></div>
            <p class="hint">Use this only when a pull request exists on GitHub.</p>
          </form>
        </div>
      </section>

      <section class="panel">
        <h2>Local Server Scan</h2>
        <form id="scan-form">
          <label for="path">Server path</label>
          <div class="repo-row"><input id="path" value="."><button id="scan-button" type="submit">Scan Path</button></div>
          <p class="hint">Available when DevInspector is running on your machine. A hosted Vercel app cannot read a visitor's laptop folders.</p>
        </form>
      </section>

      <section class="panel">
        <div class="toolbar"><h2>Findings</h2><span class="status" id="status">Ready</span></div>
        <table>
          <thead><tr><th>Severity</th><th>Rule</th><th>Line</th><th>File</th><th>Message</th></tr></thead>
          <tbody id="issues"><tr><td colspan="5">Run a scan to see results.</td></tr></tbody>
        </table>
      </section>

      <section class="rules">
        <div class="card"><h3>Dockerfile</h3><p>Detects latest tags, root containers, broad copies, secrets, and missing health checks.</p></div>
        <div class="card"><h3>Environment</h3><p>Detects secret-like values and unsafe production defaults in environment files.</p></div>
        <div class="card"><h3>Dependencies</h3><p>Detects floating or weak versions in common dependency manifests.</p></div>
      </section>

      <section class="panel">
        <h2>Raw JSON</h2>
        <pre class="json" id="raw">JSON output will appear here.</pre>
      </section>
    </main>
  </div>

  <script>
    const summary = document.querySelector('#summary');
    const issues = document.querySelector('#issues');
    const raw = document.querySelector('#raw');
    const statusText = document.querySelector('#status');

    document.querySelector('#repo-form').addEventListener('submit', event => {
      event.preventDefault();
      runRemote('/scan-repo', { repo: value('#repo-only') }, '#repo-button', 'Scanning repository...');
    });

    document.querySelector('#pr-form').addEventListener('submit', event => {
      event.preventDefault();
      runRemote('/scan-pr', { repo: value('#repo'), pr: Number(value('#pr')) }, '#pr-button', 'Scanning pull request...');
    });

    document.querySelector('#scan-form').addEventListener('submit', event => {
      event.preventDefault();
      runRemote('/scan', { path: value('#path') || '.' }, '#scan-button', 'Scanning server path...');
    });

    async function runRemote(endpoint, payload, buttonSelector, loadingText) {
      const button = document.querySelector(buttonSelector);
      button.disabled = true;
      const original = button.textContent;
      button.textContent = 'Scanning...';
      statusText.textContent = loadingText;
      issues.innerHTML = '<tr><td colspan="5">' + escapeHTML(loadingText) + '</td></tr>';
      try {
        const response = await fetch(endpoint, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        const data = await response.json();
        if (!response.ok) throw new Error(data.error || 'Scan failed');
        render(data);
        statusText.textContent = 'Scan completed';
      } catch (error) {
        summary.innerHTML = metric('Score', '-') + metric('Total issues', '-') + metric('Critical', '-') + metric('Files scanned', '-');
        issues.innerHTML = '<tr><td colspan="5">' + escapeHTML(error.message) + '</td></tr>';
        raw.textContent = error.stack || error.message;
        statusText.textContent = 'Scan failed';
      } finally {
        button.disabled = false;
        button.textContent = original;
      }
    }

    function render(data) {
      summary.innerHTML = metric('Score', data.overallScore + '/100') + metric('Total issues', data.totalIssues) + metric('Critical', data.criticalIssues) + metric('Files scanned', (data.results || []).length);
      const rows = [];
      for (const file of data.results || []) {
        if (!file.issues || file.issues.length === 0) {
          rows.push('<tr><td><span class="badge OK">OK</span></td><td>' + escapeHTML(file.fileType) + '</td><td>-</td><td>' + escapeHTML(file.filePath) + '</td><td>No issues found.</td></tr>');
          continue;
        }
        for (const issue of file.issues) {
          rows.push('<tr><td><span class="badge ' + issue.severity + '">' + issue.severity + '</span></td><td>' + escapeHTML(issue.rule) + '</td><td>' + (issue.line || '-') + '</td><td>' + escapeHTML(issue.file) + '</td><td>' + escapeHTML(issue.message) + (issue.suggestion ? '<br><strong>Suggestion:</strong> ' + escapeHTML(issue.suggestion) : '') + '</td></tr>');
        }
      }
      issues.innerHTML = rows.length ? rows.join('') : '<tr><td colspan="5">No supported files were found in this repository.</td></tr>';
      raw.textContent = JSON.stringify(data, null, 2);
    }

    function metric(label, value) {
      return '<div class="metric"><span>' + escapeHTML(label) + '</span><strong>' + escapeHTML(value) + '</strong></div>';
    }

    function value(selector) { return document.querySelector(selector).value.trim(); }

    function escapeHTML(value) {
      return String(value).replace(/[&<>'"]/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[char]));
    }
  </script>
</body>
</html>`
