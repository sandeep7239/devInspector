package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sandeep7239/devInspector/internal/rules"
	"github.com/sandeep7239/devInspector/internal/scanner"
	"github.com/sandeep7239/devInspector/internal/utils"
)

type scanRequest struct {
	Path string `json:"path"`
}

func Start(port string, logLevel string) error {
	logger := utils.NewLogger(logLevel)
	mux := http.NewServeMux()

	mux.HandleFunc("/", dashboardHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "DevInspector"})
	})
	mux.HandleFunc("/scan", scanHandler)

	addr := ":" + port
	logger.Info("DevInspector UI and API listening on http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
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
    :root { font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color: #152033; background: #f3f6fb; }
    * { box-sizing: border-box; }
    body { margin: 0; background: #f3f6fb; color: #152033; }
    .shell { min-height: 100vh; }
    .hero { background: linear-gradient(135deg, #0f172a 0%, #1d4ed8 58%, #0f766e 100%); color: white; padding: 28px min(5vw, 56px) 34px; }
    .nav { display: flex; justify-content: space-between; align-items: center; gap: 16px; margin-bottom: 36px; }
    .brand { display: flex; align-items: center; gap: 10px; font-size: 22px; font-weight: 800; }
    .brand-mark { width: 34px; height: 34px; display: grid; place-items: center; border-radius: 8px; background: rgba(255,255,255,.15); border: 1px solid rgba(255,255,255,.28); }
    .nav-links { display: flex; gap: 8px; flex-wrap: wrap; }
    .nav-links a { color: #dbeafe; text-decoration: none; font-size: 14px; padding: 8px 10px; }
    .hero-grid { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(320px, .9fr); gap: 28px; align-items: stretch; }
    .eyebrow { color: #bfdbfe; font-weight: 800; letter-spacing: 0; text-transform: uppercase; font-size: 13px; }
    h1 { font-size: clamp(36px, 5vw, 64px); line-height: 1; margin: 12px 0 16px; letter-spacing: 0; }
    .hero-copy { color: #e2e8f0; font-size: 17px; line-height: 1.6; max-width: 760px; }
    .hero-actions { display: flex; gap: 12px; flex-wrap: wrap; margin-top: 24px; }
    .button, button { border: 0; border-radius: 7px; padding: 12px 16px; font-weight: 800; cursor: pointer; text-decoration: none; display: inline-flex; align-items: center; justify-content: center; min-height: 44px; }
    .primary { background: white; color: #1d4ed8; }
    .secondary { background: rgba(255,255,255,.13); color: white; border: 1px solid rgba(255,255,255,.28); }
    .terminal { background: #09111f; border: 1px solid rgba(255,255,255,.18); border-radius: 8px; padding: 16px; box-shadow: 0 24px 70px rgba(0,0,0,.28); min-height: 310px; }
    .terminal-top { display: flex; gap: 6px; margin-bottom: 14px; }
    .dot { width: 10px; height: 10px; border-radius: 50%; background: #ef4444; } .dot:nth-child(2){background:#f59e0b}.dot:nth-child(3){background:#22c55e}
    .terminal pre { margin: 0; color: #cbd5e1; white-space: pre-wrap; line-height: 1.55; font-size: 14px; }
    main { padding: 28px min(5vw, 56px) 48px; display: grid; gap: 20px; }
    .section-title { margin: 0 0 12px; font-size: 24px; }
    .grid { display: grid; gap: 16px; }
    .cards { grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); }
    .panel, .card { background: white; border: 1px solid #d9e2ef; border-radius: 8px; box-shadow: 0 8px 24px rgba(15, 23, 42, 0.06); }
    .panel { padding: 18px; }
    .card { padding: 16px; }
    .card h3 { margin: 0 0 8px; font-size: 16px; }
    .card p { margin: 0; color: #526175; line-height: 1.5; }
    .scan-layout { display: grid; grid-template-columns: minmax(0, 1fr) 320px; gap: 16px; align-items: start; }
    form { display: grid; gap: 12px; }
    label { font-weight: 800; }
    .input-row { display: flex; gap: 10px; flex-wrap: wrap; }
    input { min-width: min(560px, 100%); flex: 1; border: 1px solid #cbd5e1; border-radius: 7px; padding: 12px 13px; font-size: 15px; }
    button { background: #2563eb; color: white; }
    button:disabled { opacity: .65; cursor: wait; }
    .hint { color: #64748b; font-size: 14px; }
    .summary { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
    .metric { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 14px; }
    .metric span { color: #64748b; font-size: 13px; font-weight: 700; }
    .metric strong { display: block; font-size: 28px; margin-top: 4px; }
    table { width: 100%; border-collapse: collapse; font-size: 14px; }
    th, td { text-align: left; border-bottom: 1px solid #e2e8f0; padding: 11px 10px; vertical-align: top; }
    th { font-size: 12px; text-transform: uppercase; color: #475569; background: #f8fafc; }
    .badge { display: inline-block; border-radius: 999px; padding: 3px 9px; font-size: 12px; font-weight: 800; background: #e2e8f0; color: #334155; }
    .CRITICAL, .ERROR { background: #fee2e2; color: #991b1b; }
    .WARNING { background: #fef3c7; color: #92400e; }
    .INFO { background: #dbeafe; color: #1e40af; }
    .OK { background: #dcfce7; color: #166534; }
    code, pre { font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace; }
    .json { white-space: pre-wrap; background: #111827; color: #d1d5db; border-radius: 8px; padding: 14px; overflow: auto; max-height: 360px; }
    .workflow { display: grid; grid-template-columns: repeat(auto-fit, minmax(230px, 1fr)); gap: 12px; }
    .step { border-left: 4px solid #2563eb; padding: 12px 14px; background: #f8fafc; border-radius: 6px; }
    .step strong { display: block; margin-bottom: 4px; }
    @media (max-width: 860px) { .hero-grid, .scan-layout { grid-template-columns: 1fr; } .nav { align-items: flex-start; flex-direction: column; } }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <nav class="nav">
        <div class="brand"><span class="brand-mark">DI</span> DevInspector</div>
        <div class="nav-links"><a href="#scan">Scan</a><a href="#rules">Rules</a><a href="#pr">PR Checks</a><a href="/health">Health</a></div>
      </nav>
      <div class="hero-grid">
        <div>
          <div class="eyebrow">Production readiness scanner</div>
          <h1>Find risky DevOps code before it reaches production.</h1>
          <p class="hero-copy">DevInspector checks repositories and pull-request branches for Dockerfile, environment, and dependency issues. Use it locally, from this dashboard, or inside GitHub Actions.</p>
          <div class="hero-actions"><a class="button primary" href="#scan">Run a Scan</a><a class="button secondary" href="#pr">Validate a PR</a></div>
        </div>
        <div class="terminal">
          <div class="terminal-top"><span class="dot"></span><span class="dot"></span><span class="dot"></span></div>
          <pre>devinspector scan --format=json .

Rules loaded:
  dockerfile-validation
  env-security
  dependency-version

Output:
  score, severity, file, line, suggestion</pre>
        </div>
      </div>
    </section>

    <main>
      <section id="scan" class="scan-layout">
        <div class="panel">
          <h2 class="section-title">Repository Scanner</h2>
          <form id="scan-form">
            <label for="path">Project path</label>
            <div class="input-row"><input id="path" name="path" value="." autocomplete="off"><button id="scan-button" type="submit">Run Scan</button></div>
            <div class="hint">Use <code>.</code> for this repo, or paste another local repo path like <code>C:\Users\Sandeep\some-project</code>.</div>
          </form>
        </div>
        <aside class="summary" id="summary">
          <div class="metric"><span>Score</span><strong>-</strong></div>
          <div class="metric"><span>Total issues</span><strong>-</strong></div>
          <div class="metric"><span>Critical</span><strong>-</strong></div>
          <div class="metric"><span>Files scanned</span><strong>-</strong></div>
        </aside>
      </section>

      <section class="panel">
        <h2 class="section-title">Findings</h2>
        <table>
          <thead><tr><th>Severity</th><th>Rule</th><th>Line</th><th>File</th><th>Message</th></tr></thead>
          <tbody id="issues"><tr><td colspan="5">Run a scan to see results.</td></tr></tbody>
        </table>
      </section>

      <section id="rules" class="grid cards">
        <div class="card"><h3>Dockerfile validation</h3><p>Flags latest tags, unpinned images, root containers, missing health checks, copied secrets, and missing .dockerignore files.</p></div>
        <div class="card"><h3>.env security</h3><p>Detects secret-like values, production debug flags, tokens, passwords, and unsafe environment defaults.</p></div>
        <div class="card"><h3>Dependency versions</h3><p>Checks Go, Node, and Python dependency manifests for floating or weak version declarations.</p></div>
      </section>

      <section id="pr" class="panel">
        <h2 class="section-title">How PR validation works</h2>
        <div class="workflow">
          <div class="step"><strong>Manual PR check</strong>Checkout the PR branch locally, then scan that folder with DevInspector.</div>
          <div class="step"><strong>Automatic PR check</strong>GitHub Actions builds DevInspector and runs it on every pull request.</div>
          <div class="step"><strong>Quality decision</strong>The scanner scores supported files and fails CI when critical findings exist.</div>
        </div>
      </section>

      <section class="panel">
        <h2 class="section-title">Raw JSON</h2>
        <pre class="json" id="raw">JSON output will appear here.</pre>
      </section>
    </main>
  </div>
  <script>
    const form = document.querySelector('#scan-form');
    const button = document.querySelector('#scan-button');
    const summary = document.querySelector('#summary');
    const issues = document.querySelector('#issues');
    const raw = document.querySelector('#raw');

    form.addEventListener('submit', async (event) => {
      event.preventDefault();
      button.disabled = true;
      button.textContent = 'Scanning...';
      issues.innerHTML = '<tr><td colspan="5">Scanning project...</td></tr>';
      try {
        const response = await fetch('/scan', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: document.querySelector('#path').value || '.' })
        });
        const data = await response.json();
        if (!response.ok) throw new Error(data.error || 'Scan failed');
        render(data);
      } catch (error) {
        summary.innerHTML = metric('Score', '-') + metric('Total issues', '-') + metric('Critical', '-') + metric('Files scanned', '-');
        issues.innerHTML = '<tr><td colspan="5">' + escapeHTML(error.message) + '</td></tr>';
        raw.textContent = error.stack || error.message;
      } finally {
        button.disabled = false;
        button.textContent = 'Run Scan';
      }
    });

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
      issues.innerHTML = rows.length ? rows.join('') : '<tr><td colspan="5">No matching files found.</td></tr>';
      raw.textContent = JSON.stringify(data, null, 2);
    }

    function metric(label, value) {
      return '<div class="metric"><span>' + escapeHTML(label) + '</span><strong>' + escapeHTML(value) + '</strong></div>';
    }

    function escapeHTML(value) {
      return String(value).replace(/[&<>'"]/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[char]));
    }
  </script>
</body>
</html>`
