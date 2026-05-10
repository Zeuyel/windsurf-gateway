package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"windsurf-gateway/internal/patcher"
)

type restoreRequest struct {
	ConfigDir  string `json:"config_dir,omitempty"`
	InstallDir string `json:"install_dir,omitempty"`
}

func main() {
	var (
		listenAddr  = flag.String("listen", "127.0.0.1:0", "listen address for the local UI server")
		noBrowser   = flag.Bool("no-browser", false, "do not auto-open the local UI in a browser")
		applyOnce   = flag.Bool("apply", false, "apply a patch once in CLI mode")
		detectOnly  = flag.Bool("detect", false, "print detected patch state as JSON")
		restoreOnly = flag.Bool("restore", false, "restore the latest backup and print JSON")
		gatewayURL  = flag.String("gateway", "", "gateway endpoint, required with -apply")
		registerURL = flag.String("register-gateway", "", "optional register gateway endpoint")
		authToken   = flag.String("auth-token", "", "optional gateway user token starting with ws-")
		mode        = flag.String("mode", string(patcher.ModeAll), "patch mode: config, extension, all")
		configDir   = flag.String("config-dir", "", "custom Windsurf config directory")
		installDir  = flag.String("install-dir", "", "custom Windsurf install directory")
	)
	flag.Parse()

	if *detectOnly {
		result, err := patcher.Detect(*configDir, *installDir)
		exitWithJSON(result, err)
		return
	}
	if *restoreOnly {
		result, err := patcher.Restore(*configDir, *installDir)
		exitWithJSON(result, err)
		return
	}
	if *applyOnce {
		result, err := patcher.Apply(patcher.ApplyOptions{
			ConfigDir:          *configDir,
			InstallDir:         *installDir,
			GatewayURL:         *gatewayURL,
			RegisterGatewayURL: *registerURL,
			AuthToken:          *authToken,
			Mode:               patcher.Mode(*mode),
		})
		exitWithJSON(result, err)
		return
	}

	server := newUIServer()
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	defer listener.Close()

	url := "http://" + listener.Addr().String()
	log.Printf("Windsurf patcher UI listening on %s", url)
	if !*noBrowser {
		go func() {
			time.Sleep(200 * time.Millisecond)
			if err := openBrowser(url); err != nil {
				log.Printf("open browser failed: %v", err)
			}
		}()
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func exitWithJSON(value any, err error) {
	payload := map[string]any{"ok": err == nil}
	if err != nil {
		payload["error"] = err.Error()
	} else {
		payload["result"] = value
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
	if err != nil {
		os.Exit(1)
	}
}

func newUIServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = indexTemplate.Execute(w, map[string]any{"DefaultMode": patcher.ModeAll})
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		result, err := patcher.Detect(r.URL.Query().Get("config_dir"), r.URL.Query().Get("install_dir"))
		writeJSON(w, result, err)
	})
	mux.HandleFunc("/api/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request patcher.ApplyOptions
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, nil, fmt.Errorf("invalid request: %w", err))
			return
		}
		result, err := patcher.Apply(request)
		writeJSON(w, result, err)
	})
	mux.HandleFunc("/api/restore", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request restoreRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, nil, fmt.Errorf("invalid request: %w", err))
			return
		}
		result, err := patcher.Restore(request.ConfigDir, request.InstallDir)
		writeJSON(w, result, err)
	})
	return &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}

func writeJSON(w http.ResponseWriter, result any, err error) {
	status := http.StatusOK
	if err != nil {
		status = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	payload := map[string]any{"ok": err == nil}
	if err != nil {
		payload["error"] = err.Error()
	} else {
		payload["result"] = result
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

var indexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Windsurf Patcher</title>
  <style>
    :root {
      --bg: linear-gradient(135deg, #f4efe7 0%, #dce7f2 52%, #f7fafc 100%);
      --panel: rgba(255,255,255,.86);
      --ink: #142235;
      --muted: #55687f;
      --line: rgba(20,34,53,.12);
      --accent: #0f6c5c;
      --accent-2: #cd5c34;
      --shadow: 0 24px 60px rgba(20,34,53,.14);
      font-family: "IBM Plex Sans", "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--ink);
      background: var(--bg);
      background-attachment: fixed;
    }
    .shell {
      width: min(1160px, calc(100vw - 32px));
      margin: 32px auto;
      display: grid;
      grid-template-columns: 420px 1fr;
      gap: 20px;
    }
    .panel {
      background: var(--panel);
      backdrop-filter: blur(16px);
      border: 1px solid var(--line);
      border-radius: 24px;
      box-shadow: var(--shadow);
      overflow: hidden;
    }
    .hero {
      padding: 28px;
      background: radial-gradient(circle at top left, rgba(15,108,92,.18), transparent 45%), radial-gradient(circle at bottom right, rgba(205,92,52,.16), transparent 40%);
    }
    .hero h1 {
      margin: 0 0 8px;
      font-size: 32px;
      line-height: 1;
      letter-spacing: -.03em;
    }
    .hero p {
      margin: 0;
      color: var(--muted);
      line-height: 1.5;
    }
    .form {
      padding: 24px 28px 28px;
      display: grid;
      gap: 14px;
    }
    label {
      font-size: 13px;
      font-weight: 700;
      letter-spacing: .04em;
      text-transform: uppercase;
      color: var(--muted);
      display: block;
      margin-bottom: 6px;
    }
    input, select, textarea, button {
      font: inherit;
    }
    input, select {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 14px;
      padding: 12px 14px;
      background: rgba(255,255,255,.92);
      color: var(--ink);
    }
    .hint {
      color: var(--muted);
      font-size: 13px;
      line-height: 1.45;
    }
    .row {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 12px;
    }
    .toggle {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 8px;
      background: rgba(20,34,53,.04);
      padding: 6px;
      border-radius: 16px;
      border: 1px solid var(--line);
    }
    .toggle button {
      border: 0;
      background: transparent;
      border-radius: 12px;
      padding: 12px;
      color: var(--muted);
      cursor: pointer;
      transition: .18s ease;
    }
    .toggle button.active {
      color: white;
      background: var(--ink);
      box-shadow: 0 12px 24px rgba(20,34,53,.22);
    }
    .actions {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
    }
    .actions button {
      border: 0;
      border-radius: 14px;
      padding: 12px 16px;
      cursor: pointer;
      font-weight: 700;
    }
    .primary { background: var(--accent); color: white; }
    .secondary { background: rgba(20,34,53,.08); color: var(--ink); }
    .danger { background: var(--accent-2); color: white; }
    .status {
      padding: 16px 18px;
      margin-top: 8px;
      border-top: 1px solid var(--line);
      background: rgba(20,34,53,.03);
      color: var(--muted);
      min-height: 64px;
    }
    .board {
      padding: 22px;
      display: grid;
      gap: 18px;
    }
    .cards {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 14px;
    }
    .card {
      background: rgba(255,255,255,.9);
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 18px;
    }
    .card h3 {
      margin: 0 0 10px;
      font-size: 13px;
      text-transform: uppercase;
      letter-spacing: .05em;
      color: var(--muted);
    }
    .metric {
      font-size: 24px;
      font-weight: 800;
      letter-spacing: -.03em;
    }
    .stack {
      display: grid;
      gap: 12px;
    }
    .kv {
      display: grid;
      grid-template-columns: 160px 1fr;
      gap: 10px;
      align-items: start;
      padding: 12px 0;
      border-top: 1px solid rgba(20,34,53,.08);
    }
    .kv:first-child { border-top: 0; padding-top: 0; }
    .kv code {
      word-break: break-all;
      color: var(--ink);
      background: rgba(20,34,53,.05);
      padding: 2px 6px;
      border-radius: 8px;
    }
    .list {
      margin: 0;
      padding-left: 18px;
      color: var(--muted);
    }
    .hidden { display: none; }
    @media (max-width: 980px) {
      .shell { grid-template-columns: 1fr; }
      .cards { grid-template-columns: 1fr; }
      .row { grid-template-columns: 1fr; }
      .kv { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="panel">
      <div class="hero">
        <h1>Windsurf Patcher</h1>
        <p>Patch Windsurf to your gateway, keep a stable local client identity, and optionally attach a gateway user token for per-user backend routing. Close Windsurf before Apply or Restore.</p>
      </div>
      <div class="form">
        <div>
          <label for="gateway">Gateway Endpoint</label>
          <input id="gateway" placeholder="https://gateway.example.com">
          <div class="hint">Point this at the gateway root, not <code>/proxy</code>.</div>
        </div>
        <div>
          <label>Routing Mode</label>
          <div class="toggle">
            <button id="mode-anon" class="active" type="button">Anonymous Sticky</button>
            <button id="mode-user" type="button">Gateway User Token</button>
          </div>
          <div class="hint">Anonymous mode uses a per-client placeholder key. Gateway-user mode sends your <code>ws-...</code> token only to the gateway, never upstream.</div>
        </div>
        <div id="auth-block" class="hidden">
          <label for="auth-token">Gateway User Token</label>
          <input id="auth-token" placeholder="ws-...">
        </div>
        <div class="row">
          <div>
            <label for="patch-mode">Patch Mode</label>
            <select id="patch-mode">
              <option value="all">Full Patch</option>
              <option value="config">Config + Global State</option>
              <option value="extension">Extension Only</option>
            </select>
          </div>
          <div>
            <label for="register-gateway">Register Endpoint</label>
            <input id="register-gateway" placeholder="optional">
          </div>
        </div>
        <div class="row">
          <div>
            <label for="config-dir">Config Directory</label>
            <input id="config-dir" placeholder="auto detect">
          </div>
          <div>
            <label for="install-dir">Install Directory</label>
            <input id="install-dir" placeholder="auto detect">
          </div>
        </div>
        <div class="actions">
          <button class="primary" id="apply">Apply Patch</button>
          <button class="secondary" id="refresh">Refresh Detect</button>
          <button class="danger" id="restore">Restore Last Backup</button>
        </div>
      </div>
      <div class="status" id="status">Ready.</div>
    </section>

    <section class="panel board">
      <div class="cards">
        <div class="card">
          <h3>Current Gateway</h3>
          <div class="metric" id="metric-gateway">-</div>
        </div>
        <div class="card">
          <h3>Auth Mode</h3>
          <div class="metric" id="metric-auth">-</div>
        </div>
        <div class="card">
          <h3>Patch State</h3>
          <div class="metric" id="metric-patch">-</div>
        </div>
      </div>

      <div class="stack">
        <div class="card">
          <h3>Detected Paths</h3>
          <div class="kv"><strong>Config Dir</strong><span id="path-config">-</span></div>
          <div class="kv"><strong>Install Dir</strong><span id="path-install">-</span></div>
          <div class="kv"><strong>Settings</strong><code id="path-settings">-</code></div>
          <div class="kv"><strong>Global State</strong><code id="path-state">-</code></div>
          <div class="kv"><strong>Extension</strong><code id="path-extension">-</code></div>
        </div>
        <div class="card">
          <h3>Patch Health</h3>
          <div class="kv"><strong>Settings</strong><span id="health-settings">-</span></div>
          <div class="kv"><strong>Global State</strong><span id="health-global">-</span></div>
          <div class="kv"><strong>Extension</strong><span id="health-extension">-</span></div>
          <div class="kv"><strong>Placeholder Key</strong><span id="health-placeholder">-</span></div>
        </div>
        <div class="card">
          <h3>Messages</h3>
          <ul class="list" id="messages"></ul>
        </div>
      </div>
    </section>
  </div>

  <script>
    const state = { authMode: 'anonymous' }
    const $ = (id) => document.getElementById(id)
    const status = $('status')
    const messages = $('messages')

    function setStatus(text, isError = false) {
      status.textContent = text
      status.style.color = isError ? '#b42318' : 'var(--muted)'
    }

    function setAuthMode(mode) {
      state.authMode = mode
      $('mode-anon').classList.toggle('active', mode === 'anonymous')
      $('mode-user').classList.toggle('active', mode === 'gateway-user')
      $('auth-block').classList.toggle('hidden', mode !== 'gateway-user')
    }

    async function request(url, options) {
      const response = await fetch(url, options)
      const payload = await response.json()
      if (!payload.ok) throw new Error(payload.error || 'request failed')
      return payload.result
    }

    function currentQuery() {
      const params = new URLSearchParams()
      if ($('config-dir').value.trim()) params.set('config_dir', $('config-dir').value.trim())
      if ($('install-dir').value.trim()) params.set('install_dir', $('install-dir').value.trim())
      const query = params.toString()
      return query ? '?' + query : ''
    }

    function renderMessages(items) {
      messages.innerHTML = ''
      ;(items || []).forEach((item) => {
        const li = document.createElement('li')
        li.textContent = item
        messages.appendChild(li)
      })
      if (!messages.children.length) {
        const li = document.createElement('li')
        li.textContent = 'No extra messages.'
        messages.appendChild(li)
      }
    }

    function render(result) {
      $('path-config').textContent = result.environment.config_dir || '-'
      $('path-install').textContent = result.environment.install_dir || '-'
      $('path-settings').textContent = result.environment.settings_path || '-'
      $('path-state').textContent = result.environment.global_state_path || '-'
      $('path-extension').textContent = result.environment.extension_path || '-'

      $('metric-gateway').textContent = result.settings.gateway_url || '-'
      $('metric-auth').textContent = result.global_state.auth_mode || result.extension.auth_mode || '-'
      $('metric-patch').textContent = result.patch_state_mode || 'missing'

      $('health-settings').textContent = result.settings.exists ? (result.settings.devin_cloud_disabled ? 'gateway endpoint + devin-cloud off' : 'settings found') : 'missing'
      $('health-global').textContent = result.global_state.exists ? (result.global_state.auth_token_summary || 'present') : (result.global_state.read_error || 'missing')
      $('health-extension').textContent = result.extension.exists ? (result.extension.has_auth_fallback ? 'fallback ready' : 'no fallback yet') : (result.extension.read_error || 'missing')
      $('health-placeholder').textContent = result.patch_state_placeholder || 'missing'

      if (!$('gateway').value && result.settings.gateway_url) $('gateway').value = result.settings.gateway_url
      if (result.global_state.auth_mode === 'gateway-user' || result.extension.auth_mode === 'gateway-user') setAuthMode('gateway-user')
      renderMessages(result.messages)
    }

    async function refreshDetect(message = 'Detection refreshed.') {
      setStatus('Detecting...')
      try {
        const result = await request('/api/state' + currentQuery(), { method: 'GET' })
        render(result)
        setStatus(message)
      } catch (error) {
        setStatus(error.message, true)
      }
    }

    async function applyPatch() {
      setStatus('Applying patch...')
      const payload = {
        gateway_url: $('gateway').value.trim(),
        register_gateway_url: $('register-gateway').value.trim(),
        config_dir: $('config-dir').value.trim(),
        install_dir: $('install-dir').value.trim(),
        mode: $('patch-mode').value,
      }
      if (state.authMode === 'gateway-user') payload.auth_token = $('auth-token').value.trim()
      try {
        const result = await request('/api/apply', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        })
        render(result.detect)
        renderMessages(['Backup: ' + (result.backup_dir || 'none'), 'Effective auth mode: ' + (result.effective_auth_mode || '-'), ...(result.messages || []), ...(result.detect.messages || [])])
        setStatus('Patch applied. Restart Windsurf.')
      } catch (error) {
        setStatus(error.message, true)
      }
    }

    async function restoreBackup() {
      setStatus('Restoring latest backup...')
      try {
        const result = await request('/api/restore', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            config_dir: $('config-dir').value.trim(),
            install_dir: $('install-dir').value.trim(),
          }),
        })
        render(result.detect)
        renderMessages(['Restored from: ' + result.restored_from, ...(result.messages || []), ...(result.detect.messages || [])])
        setStatus('Latest backup restored. Restart Windsurf.')
      } catch (error) {
        setStatus(error.message, true)
      }
    }

    $('mode-anon').addEventListener('click', () => setAuthMode('anonymous'))
    $('mode-user').addEventListener('click', () => setAuthMode('gateway-user'))
    $('apply').addEventListener('click', applyPatch)
    $('refresh').addEventListener('click', () => refreshDetect())
    $('restore').addEventListener('click', restoreBackup)

    refreshDetect('Detected current Windsurf patch state.')
  </script>
</body>
</html>`))
