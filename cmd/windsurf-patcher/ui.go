package main

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Windsurf Patch 向导</title>
  <style>
    :root {
      --bg:
        radial-gradient(circle at 10% 20%, rgba(17, 110, 102, .18), transparent 32%),
        radial-gradient(circle at 90% 12%, rgba(210, 95, 50, .18), transparent 28%),
        linear-gradient(150deg, #f2ede4 0%, #e6efe9 48%, #eef4f8 100%);
      --panel: rgba(255, 255, 255, .88);
      --panel-strong: rgba(255, 255, 255, .96);
      --ink: #17263a;
      --muted: #63758e;
      --line: rgba(23, 38, 58, .11);
      --line-strong: rgba(23, 38, 58, .2);
      --accent: #116e66;
      --accent-strong: #0c5a53;
      --accent-soft: rgba(17, 110, 102, .1);
      --warn: #c65e31;
      --warn-soft: rgba(198, 94, 49, .12);
      --ok: #1b7f43;
      --ok-soft: rgba(27, 127, 67, .12);
      --shadow: 0 22px 60px rgba(23, 38, 58, .12);
      --radius-lg: 28px;
      --radius-md: 20px;
      --radius-sm: 14px;
      font-family: "Noto Sans SC", "PingFang SC", "Microsoft YaHei", "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--ink);
      background: var(--bg);
      background-attachment: fixed;
    }
    button, input, select {
      font: inherit;
    }
    .page {
      width: min(1320px, calc(100vw - 36px));
      margin: 28px auto;
      display: grid;
      grid-template-columns: minmax(0, 1.12fr) minmax(340px, .88fr);
      gap: 22px;
    }
    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: var(--radius-lg);
      box-shadow: var(--shadow);
      backdrop-filter: blur(18px);
      overflow: hidden;
    }
    .wizard {
      padding: 28px;
      display: grid;
      gap: 18px;
    }
    .hero {
      padding: 28px;
      border-radius: 24px;
      background:
        linear-gradient(135deg, rgba(17, 110, 102, .92), rgba(12, 60, 89, .92)),
        linear-gradient(135deg, rgba(255,255,255,.16), rgba(255,255,255,0));
      color: #fff;
      position: relative;
      overflow: hidden;
    }
    .hero::after {
      content: "";
      position: absolute;
      inset: auto -36px -42px auto;
      width: 220px;
      height: 220px;
      border-radius: 999px;
      background: rgba(255,255,255,.08);
      filter: blur(4px);
    }
    .hero-top {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 16px;
      position: relative;
      z-index: 1;
    }
    .hero h1 {
      margin: 0;
      font-size: clamp(30px, 4vw, 40px);
      line-height: 1.02;
      letter-spacing: -.04em;
    }
    .hero p {
      margin: 12px 0 0;
      max-width: 720px;
      line-height: 1.7;
      color: rgba(255,255,255,.84);
    }
    .chip-row {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      margin-top: 18px;
      position: relative;
      z-index: 1;
    }
    .chip {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 9px 12px;
      border-radius: 999px;
      background: rgba(255,255,255,.12);
      border: 1px solid rgba(255,255,255,.18);
      font-size: 13px;
      color: rgba(255,255,255,.92);
    }
    .hero-badge {
      flex: none;
      padding: 10px 14px;
      border-radius: 16px;
      background: rgba(255,255,255,.14);
      border: 1px solid rgba(255,255,255,.18);
      font-size: 12px;
      letter-spacing: .08em;
      text-transform: uppercase;
      color: rgba(255,255,255,.88);
    }
    .notice {
      display: grid;
      gap: 8px;
      padding: 18px 20px;
      border-radius: 20px;
      background: var(--warn-soft);
      border: 1px solid rgba(198, 94, 49, .18);
    }
    .notice strong {
      font-size: 15px;
    }
    .notice span {
      color: #7a4a33;
      line-height: 1.6;
      font-size: 14px;
    }
    .step-grid {
      display: grid;
      gap: 16px;
    }
    .step-card {
      padding: 22px;
      border-radius: 24px;
      border: 1px solid var(--line);
      background: var(--panel-strong);
      display: grid;
      gap: 16px;
    }
    .step-head {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 16px;
    }
    .step-title {
      display: flex;
      align-items: flex-start;
      gap: 14px;
      min-width: 0;
    }
    .step-index {
      width: 38px;
      height: 38px;
      border-radius: 14px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      font-weight: 800;
      background: var(--accent-soft);
      color: var(--accent);
      flex: none;
    }
    .step-head h2 {
      margin: 0;
      font-size: 20px;
      letter-spacing: -.02em;
    }
    .step-head p {
      margin: 6px 0 0;
      color: var(--muted);
      line-height: 1.65;
      font-size: 14px;
    }
    .badge {
      flex: none;
      padding: 9px 12px;
      border-radius: 999px;
      font-size: 12px;
      font-weight: 700;
      border: 1px solid var(--line);
      color: var(--muted);
      background: rgba(99, 117, 142, .08);
    }
    .badge-ready {
      background: var(--ok-soft);
      color: var(--ok);
      border-color: rgba(27, 127, 67, .18);
    }
    .badge-active {
      background: var(--accent-soft);
      color: var(--accent);
      border-color: rgba(17, 110, 102, .18);
    }
    .badge-warn {
      background: var(--warn-soft);
      color: var(--warn);
      border-color: rgba(198, 94, 49, .18);
    }
    .field-grid {
      display: grid;
      gap: 14px;
    }
    .field-row {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 14px;
    }
    .field label {
      display: block;
      margin-bottom: 8px;
      font-size: 13px;
      font-weight: 700;
      color: var(--muted);
      letter-spacing: .02em;
    }
    .field input,
    .field select {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: 13px 14px;
      background: #fff;
      color: var(--ink);
      outline: none;
      transition: border-color .18s ease, box-shadow .18s ease;
    }
    .field input:focus,
    .field select:focus {
      border-color: rgba(17, 110, 102, .45);
      box-shadow: 0 0 0 4px rgba(17, 110, 102, .1);
    }
    .field-hint {
      margin-top: 8px;
      color: var(--muted);
      line-height: 1.55;
      font-size: 13px;
    }
    .mode-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 12px;
    }
    .mode-card {
      border: 1px solid var(--line);
      border-radius: 20px;
      padding: 18px;
      background: rgba(255,255,255,.84);
      cursor: pointer;
      transition: transform .18s ease, border-color .18s ease, box-shadow .18s ease, background .18s ease;
      text-align: left;
    }
    .mode-card:hover {
      transform: translateY(-1px);
      border-color: rgba(17, 110, 102, .3);
      box-shadow: 0 14px 28px rgba(23, 38, 58, .08);
    }
    .mode-card.active {
      background: linear-gradient(180deg, rgba(17, 110, 102, .12), rgba(17, 110, 102, .04));
      border-color: rgba(17, 110, 102, .36);
    }
    .mode-card strong {
      display: block;
      font-size: 16px;
      margin-bottom: 8px;
    }
    .mode-card span {
      display: block;
      color: var(--muted);
      line-height: 1.6;
      font-size: 14px;
    }
    .mode-tag {
      margin-top: 10px;
      display: inline-flex;
      align-items: center;
      padding: 6px 10px;
      border-radius: 999px;
      font-size: 12px;
      font-weight: 700;
      background: rgba(23, 38, 58, .06);
      color: var(--muted);
    }
    .inline-note {
      padding: 14px 16px;
      border-radius: 18px;
      background: rgba(23, 38, 58, .04);
      color: var(--muted);
      line-height: 1.6;
      font-size: 14px;
    }
    .path-grid {
      display: grid;
      gap: 10px;
    }
    .path-item {
      padding: 12px 14px;
      border-radius: 16px;
      background: rgba(23, 38, 58, .04);
      border: 1px solid rgba(23, 38, 58, .05);
    }
    .path-item strong {
      display: block;
      font-size: 12px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: .06em;
      margin-bottom: 6px;
    }
    .path-item code,
    .path-item span {
      color: var(--ink);
      word-break: break-all;
      line-height: 1.5;
    }
    .action-row {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
    }
    .action-row button {
      border: 0;
      border-radius: 16px;
      padding: 14px 18px;
      font-weight: 700;
      cursor: pointer;
      transition: transform .16s ease, filter .16s ease;
    }
    .action-row button:hover {
      transform: translateY(-1px);
      filter: brightness(.98);
    }
    .btn-primary {
      background: var(--accent);
      color: #fff;
    }
    .btn-secondary {
      background: rgba(23, 38, 58, .08);
      color: var(--ink);
    }
    .btn-danger {
      background: var(--warn);
      color: #fff;
    }
    .status-card {
      padding: 16px 18px;
      border-radius: 18px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,.92);
      min-height: 74px;
      display: grid;
      gap: 6px;
      align-content: center;
    }
    .status-label {
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .08em;
      color: var(--muted);
    }
    .status-text {
      font-size: 15px;
      line-height: 1.6;
      color: var(--ink);
    }
    .status-error .status-text {
      color: #a63c1f;
    }
    .board {
      padding: 28px;
      display: grid;
      gap: 16px;
      align-content: start;
    }
    .board-hero {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 12px;
    }
    .board-hero h3 {
      margin: 0;
      font-size: 24px;
      letter-spacing: -.03em;
    }
    .board-hero p {
      margin: 8px 0 0;
      color: var(--muted);
      line-height: 1.6;
      font-size: 14px;
    }
    .summary-grid {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 12px;
    }
    .summary-card {
      padding: 18px;
      border-radius: 20px;
      border: 1px solid var(--line);
      background: var(--panel-strong);
    }
    .summary-card strong {
      display: block;
      font-size: 12px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: .06em;
      margin-bottom: 10px;
    }
    .summary-value {
      font-size: 22px;
      font-weight: 800;
      letter-spacing: -.03em;
      line-height: 1.2;
      word-break: break-word;
    }
    .sidebar-card {
      padding: 20px;
      border-radius: 22px;
      border: 1px solid var(--line);
      background: var(--panel-strong);
      display: grid;
      gap: 14px;
    }
    .sidebar-card h4 {
      margin: 0;
      font-size: 18px;
      letter-spacing: -.02em;
    }
    .sidebar-card p {
      margin: 0;
      color: var(--muted);
      line-height: 1.6;
      font-size: 14px;
    }
    .checklist {
      display: grid;
      gap: 10px;
      margin: 0;
      padding: 0;
      list-style: none;
    }
    .check-item {
      display: flex;
      align-items: flex-start;
      gap: 10px;
      padding: 10px 0;
      border-top: 1px solid rgba(23, 38, 58, .08);
    }
    .check-item:first-child {
      border-top: 0;
      padding-top: 0;
    }
    .check-mark {
      flex: none;
      width: 24px;
      height: 24px;
      border-radius: 999px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      background: rgba(23, 38, 58, .08);
      color: var(--muted);
      font-size: 12px;
      font-weight: 700;
      margin-top: 1px;
    }
    .check-item.ready .check-mark {
      background: var(--ok-soft);
      color: var(--ok);
    }
    .check-item.active .check-mark {
      background: var(--accent-soft);
      color: var(--accent);
    }
    .check-copy strong {
      display: block;
      font-size: 14px;
      margin-bottom: 4px;
    }
    .check-copy span {
      display: block;
      color: var(--muted);
      line-height: 1.55;
      font-size: 13px;
    }
    .detail-grid {
      display: grid;
      gap: 10px;
    }
    .detail-row {
      display: grid;
      grid-template-columns: 150px 1fr;
      gap: 12px;
      align-items: start;
      padding: 12px 0;
      border-top: 1px solid rgba(23, 38, 58, .08);
    }
    .detail-row:first-child {
      border-top: 0;
      padding-top: 0;
    }
    .detail-row strong {
      font-size: 13px;
      color: var(--muted);
      letter-spacing: .02em;
    }
    .detail-row code,
    .detail-row span {
      color: var(--ink);
      line-height: 1.6;
      word-break: break-word;
    }
    .message-list {
      margin: 0;
      padding-left: 18px;
      color: var(--muted);
      display: grid;
      gap: 8px;
    }
    .hidden {
      display: none;
    }
    @media (max-width: 1120px) {
      .page {
        grid-template-columns: 1fr;
      }
      .summary-grid {
        grid-template-columns: 1fr;
      }
    }
    @media (max-width: 760px) {
      .wizard, .board {
        padding: 18px;
      }
      .hero, .step-card, .sidebar-card {
        padding: 18px;
      }
      .field-row,
      .mode-grid {
        grid-template-columns: 1fr;
      }
      .detail-row {
        grid-template-columns: 1fr;
      }
      .board-hero {
        flex-direction: column;
      }
    }
  </style>
</head>
<body>
  <div class="page">
    <section class="panel wizard">
      <div class="hero">
        <div class="hero-top">
          <div>
            <h1>Windsurf Patch 向导</h1>
            <p>把 Windsurf 安全接入你的 Gateway。界面按步骤推进，用户只需要检测、选择模式、填写地址，然后一键应用。</p>
          </div>
          <div class="hero-badge">Local Wizard</div>
        </div>
        <div class="chip-row">
          <span class="chip">本地页面，不上传配置</span>
          <span class="chip">支持匿名粘性 / 用户分发</span>
          <span class="chip">可恢复最近一次备份</span>
        </div>
      </div>

      <div class="notice">
        <strong>开始前先完全退出 Windsurf</strong>
        <span>这样 <code>state.vscdb</code> 和内置客户端脚本 <code>extension.js</code> 不会被占用，应用 patch 或恢复备份时更稳定。</span>
      </div>

      <div class="step-grid">
        <section class="step-card">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">1</div>
              <div>
                <h2>检测本机 Windsurf</h2>
                <p>自动识别配置目录、安装目录和当前 patch 状态。先点一次检测，确认本机路径没有偏。</p>
              </div>
            </div>
            <span class="badge" id="badge-detect">待检测</span>
          </div>
          <div class="action-row">
            <button class="btn-secondary" id="refresh" type="button">重新检测</button>
          </div>
          <div class="path-grid">
            <div class="path-item">
              <strong>配置目录</strong>
              <span id="path-config">-</span>
            </div>
            <div class="path-item">
              <strong>安装目录</strong>
              <span id="path-install">-</span>
            </div>
          </div>
        </section>

        <section class="step-card">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">2</div>
              <div>
                <h2>选择接入模式</h2>
                <p>如果只想匿名稳定分配后端账号，选匿名粘性。若要按网关用户分发，填入 <code>ws-...</code> 令牌。</p>
              </div>
            </div>
            <span class="badge badge-active" id="badge-route">匿名粘性</span>
          </div>
          <div class="mode-grid">
            <button id="mode-anon" class="mode-card active" type="button">
              <strong>匿名粘性</strong>
              <span>每台客户端生成独立占位 key，只进 Gateway，不把真实身份往上游透传。</span>
              <span class="mode-tag">推荐给大多数部署</span>
            </button>
            <button id="mode-user" class="mode-card" type="button">
              <strong>网关用户令牌</strong>
              <span>用户先用 <code>ws-...</code> 令牌接入 Gateway，再由 Gateway 按用户做分发、额度和策略控制。</span>
              <span class="mode-tag">适合多人共享网关</span>
            </button>
          </div>
          <div id="auth-block" class="field hidden">
            <label for="auth-token">Gateway 用户令牌</label>
            <input id="auth-token" placeholder="ws-xxxxxxxx">
            <div class="field-hint">这个令牌只会发送给你的 Gateway，不会上送到 Windsurf 上游。</div>
          </div>
        </section>

        <section class="step-card">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">3</div>
              <div>
                <h2>填写网关地址</h2>
                <p>填 Gateway 根地址，不要填 <code>/proxy</code>。如有单独的 register 地址，也可以一并填入。</p>
              </div>
            </div>
            <span class="badge" id="badge-gateway">待填写</span>
          </div>
          <div class="field-grid">
            <div class="field">
              <label for="gateway">Gateway 地址</label>
              <input id="gateway" placeholder="https://gateway.example.com">
              <div class="field-hint">客户端最终会把 <code>codeium.apiServerUrl</code> 指向这里。</div>
            </div>
            <div class="field-row">
              <div class="field">
                <label for="register-gateway">Register 地址</label>
                <input id="register-gateway" placeholder="可选">
                <div class="field-hint">不填则只改主 Gateway 地址。</div>
              </div>
              <div class="field">
                <label for="patch-mode">改写范围</label>
                <select id="patch-mode">
                  <option value="all">完整改写：配置 + 全局状态 + 客户端脚本</option>
                  <option value="config">仅配置和全局状态</option>
                  <option value="extension">仅客户端脚本</option>
                </select>
                <div class="field-hint">第一次接入建议选完整 patch。</div>
              </div>
            </div>
          </div>
        </section>

        <section class="step-card">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">4</div>
              <div>
                <h2>高级路径设置</h2>
                <p>默认会自动识别路径。只有你用了便携版、自定义安装目录或多套配置时，才需要手动填写。</p>
              </div>
            </div>
            <span class="badge" id="badge-advanced">默认自动识别</span>
          </div>
          <div class="field-row">
            <div class="field">
              <label for="config-dir">自定义配置目录</label>
              <input id="config-dir" placeholder="留空即自动检测">
            </div>
            <div class="field">
              <label for="install-dir">自定义安装目录</label>
              <input id="install-dir" placeholder="留空即自动检测">
            </div>
          </div>
          <div class="inline-note">如果检测结果已经正确，这一步可以完全跳过，不会影响应用 patch。</div>
        </section>

        <section class="step-card">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">5</div>
              <div>
                <h2>执行 patch 或恢复</h2>
                <p>检查完上面的内容后，直接应用。若要回退到上一个版本，可以恢复最近一次自动备份。</p>
              </div>
            </div>
            <span class="badge" id="badge-run">等待前置步骤</span>
          </div>
          <div class="action-row">
            <button class="btn-primary" id="apply" type="button">一键应用 Patch</button>
            <button class="btn-danger" id="restore" type="button">恢复最近一次备份</button>
          </div>
          <div class="status-card" id="status-card">
            <div class="status-label">当前状态</div>
            <div class="status-text" id="status-text">正在准备检测本机环境。</div>
          </div>
        </section>
      </div>
    </section>

    <section class="panel board">
      <div class="board-hero">
        <div>
          <h3>当前摘要</h3>
          <p>右侧会实时显示本机检测结果、当前接入模式和 patch 健康度，方便你在执行前快速确认。</p>
        </div>
        <span class="badge badge-active" id="guide-badge">向导准备中</span>
      </div>

      <div class="summary-grid">
        <div class="summary-card">
          <strong>当前 Gateway</strong>
          <div class="summary-value" id="metric-gateway">-</div>
        </div>
        <div class="summary-card">
          <strong>当前鉴权模式</strong>
          <div class="summary-value" id="metric-auth">-</div>
        </div>
        <div class="summary-card">
          <strong>本地 Patch 状态</strong>
          <div class="summary-value" id="metric-patch">-</div>
        </div>
      </div>

      <div class="sidebar-card">
        <h4>一步一步看这里</h4>
        <p>页面会根据你的输入自动告诉你下一步该做什么，不需要自己记流程。</p>
        <ul class="checklist" id="guide-list"></ul>
      </div>

      <div class="sidebar-card">
        <h4>路径与文件</h4>
        <div class="detail-grid">
          <div class="detail-row"><strong>settings.json</strong><code id="path-settings">-</code></div>
          <div class="detail-row"><strong>state.vscdb</strong><code id="path-state">-</code></div>
          <div class="detail-row"><strong>客户端脚本</strong><code id="path-extension">-</code></div>
        </div>
      </div>

      <div class="sidebar-card">
        <h4>Patch 健康状态</h4>
        <div class="detail-grid">
          <div class="detail-row"><strong>配置状态</strong><span id="health-settings">-</span></div>
          <div class="detail-row"><strong>全局状态</strong><span id="health-global">-</span></div>
          <div class="detail-row"><strong>客户端脚本兜底</strong><span id="health-extension">-</span></div>
          <div class="detail-row"><strong>本地占位 Key</strong><span id="health-placeholder">-</span></div>
        </div>
      </div>

      <div class="sidebar-card">
        <h4>执行消息</h4>
        <ul class="message-list" id="messages"></ul>
      </div>
    </section>
  </div>

  <script>
    const state = {
      authMode: "anonymous",
      detect: null
    }

    const $ = function(id) { return document.getElementById(id) }
    const statusCard = $("status-card")
    const statusText = $("status-text")
    const messages = $("messages")
    const guideList = $("guide-list")

    function setStatus(text, isError) {
      statusText.textContent = text
      statusCard.classList.toggle("status-error", !!isError)
    }

    function setBadge(id, text, tone) {
      const el = $(id)
      el.textContent = text
      el.className = "badge"
      if (tone) {
        el.classList.add("badge-" + tone)
      }
    }

    function currentGatewayValue() {
      return $("gateway").value.trim()
    }

    function currentAuthTokenValue() {
      return $("auth-token").value.trim()
    }

    function hasDetectResult() {
      return !!(state.detect && state.detect.environment)
    }

    function setAuthMode(mode) {
      state.authMode = mode
      $("mode-anon").classList.toggle("active", mode === "anonymous")
      $("mode-user").classList.toggle("active", mode === "gateway-user")
      $("auth-block").classList.toggle("hidden", mode !== "gateway-user")
      refreshWizardState()
    }

    async function request(url, options) {
      const response = await fetch(url, options)
      const payload = await response.json()
      if (!payload.ok) {
        throw new Error(payload.error || "请求失败")
      }
      return payload.result
    }

    function currentQuery() {
      const params = new URLSearchParams()
      if ($("config-dir").value.trim()) params.set("config_dir", $("config-dir").value.trim())
      if ($("install-dir").value.trim()) params.set("install_dir", $("install-dir").value.trim())
      const query = params.toString()
      return query ? "?" + query : ""
    }

    function textOrDash(value) {
      return value && String(value).trim() ? String(value) : "-"
    }

    function renderMessages(items) {
      messages.innerHTML = ""
      ;(items || []).forEach(function(item) {
        const li = document.createElement("li")
        li.textContent = item
        messages.appendChild(li)
      })
      if (!messages.children.length) {
        const li = document.createElement("li")
        li.textContent = "暂无额外消息。"
        messages.appendChild(li)
      }
    }

    function displayAuthMode(result) {
      const mode = (result.global_state && result.global_state.auth_mode) || (result.extension && result.extension.auth_mode) || ""
      switch (mode) {
        case "gateway-user":
          return "网关用户令牌"
        case "per-client-placeholder":
          return "匿名粘性"
        case "legacy-shared-placeholder":
          return "旧版共享占位"
        case "custom":
          return "自定义令牌"
        case "none":
          return "未注入"
        default:
          return mode || "未检测"
      }
    }

    function displayPatchState(result) {
      switch (result.patch_state_mode) {
        case "gateway-user":
          return "用户分发"
        case "per-client-placeholder":
          return "匿名粘性"
        case "legacy-shared-placeholder":
          return "旧版占位"
        case "none":
        case "":
          return result.patch_state_exists ? "已存在但未识别" : "未创建"
        default:
          return result.patch_state_mode || "未创建"
      }
    }

    function describeSettings(result) {
      if (!result.settings.exists) return "未找到 settings.json"
      if (result.settings.gateway_url && result.settings.devin_cloud_disabled) return "已写入 Gateway，且已关闭 devin-cloud"
      if (result.settings.gateway_url) return "已写入 Gateway"
      return "已找到 settings.json，但尚未写入 Gateway"
    }

    function describeGlobal(result) {
      if (!result.global_state.exists) return result.global_state.read_error || "未找到 state.vscdb"
      if (result.global_state.auth_token_summary) return "已注入认证状态：" + result.global_state.auth_token_summary
      if (result.global_state.onboarding_patched || result.global_state.education_patched) return "已写入 onboarding/教育状态"
      return "已找到 state.vscdb，但未看到认证兜底"
    }

    function describeExtension(result) {
      if (!result.extension.exists) return result.extension.read_error || "未找到内置客户端脚本 extension.js"
      if (result.extension.has_auth_fallback && result.extension.has_user_status_fallback) return "登录兜底和用户状态兜底都已存在"
      if (result.extension.has_auth_fallback) return "仅存在登录兜底，还未写入用户状态兜底"
      return "尚未注入客户端脚本兜底"
    }

    function render(result) {
      state.detect = result

      $("path-config").textContent = textOrDash(result.environment.config_dir)
      $("path-install").textContent = textOrDash(result.environment.install_dir)
      $("path-settings").textContent = textOrDash(result.environment.settings_path)
      $("path-state").textContent = textOrDash(result.environment.global_state_path)
      $("path-extension").textContent = textOrDash(result.environment.extension_path)

      $("metric-gateway").textContent = textOrDash(result.settings.gateway_url)
      $("metric-auth").textContent = displayAuthMode(result)
      $("metric-patch").textContent = displayPatchState(result)

      $("health-settings").textContent = describeSettings(result)
      $("health-global").textContent = describeGlobal(result)
      $("health-extension").textContent = describeExtension(result)
      $("health-placeholder").textContent = result.patch_state_placeholder || "未生成"

      if (!$("gateway").value && result.settings.gateway_url) {
        $("gateway").value = result.settings.gateway_url
      }
      if (!$("register-gateway").value && result.settings.register_gateway_url) {
        $("register-gateway").value = result.settings.register_gateway_url
      }
      if (result.global_state.auth_mode === "gateway-user" || result.extension.auth_mode === "gateway-user") {
        setAuthMode("gateway-user")
      }
      renderMessages(result.messages)
      refreshWizardState()
    }

    function renderGuide(items) {
      guideList.innerHTML = ""
      items.forEach(function(item) {
        const li = document.createElement("li")
        li.className = "check-item " + item.tone

        const mark = document.createElement("span")
        mark.className = "check-mark"
        mark.textContent = item.tone === "ready" ? "✓" : item.tone === "active" ? "→" : "·"

        const copy = document.createElement("div")
        copy.className = "check-copy"

        const title = document.createElement("strong")
        title.textContent = item.title
        const desc = document.createElement("span")
        desc.textContent = item.desc

        copy.appendChild(title)
        copy.appendChild(desc)
        li.appendChild(mark)
        li.appendChild(copy)
        guideList.appendChild(li)
      })
    }

    function refreshWizardState() {
      const detectReady = hasDetectResult()
      const gatewayReady = !!currentGatewayValue()
      const userModeReady = state.authMode !== "gateway-user" || !!currentAuthTokenValue()
      const advancedCustomized = !!$("config-dir").value.trim() || !!$("install-dir").value.trim()
      const readyToApply = gatewayReady && userModeReady

      setBadge("badge-detect", detectReady ? "已识别" : "待检测", detectReady ? "ready" : "warn")
      setBadge("badge-route", state.authMode === "gateway-user" ? "用户令牌模式" : "匿名粘性模式", "active")
      setBadge("badge-gateway", readyToApply ? "可执行" : "待填写", readyToApply ? "ready" : "warn")
      setBadge("badge-advanced", advancedCustomized ? "使用自定义路径" : "默认自动识别", advancedCustomized ? "active" : "")
      setBadge("badge-run", readyToApply ? "可以应用 Patch" : "等待前置步骤", readyToApply ? "ready" : "warn")

      let guideTone = "badge-warn"
      let guideTitle = "需要先完成前置步骤"
      if (readyToApply) {
        guideTone = "badge-ready"
        guideTitle = "可以直接应用 Patch"
      } else if (detectReady || gatewayReady) {
        guideTone = "badge-active"
        guideTitle = "正在推进配置"
      }
      $("guide-badge").className = "badge " + guideTone
      $("guide-badge").textContent = guideTitle

      const guideItems = [
        {
          tone: detectReady ? "ready" : "active",
          title: detectReady ? "本机 Windsurf 已识别" : "先检测本机环境",
          desc: detectReady ? "已拿到配置目录和安装目录，可以继续下一步。" : "点击“重新检测”，先确认本机路径和现有 patch 状态。"
        },
        {
          tone: userModeReady ? "ready" : "active",
          title: state.authMode === "gateway-user" ? "当前模式：网关用户令牌" : "当前模式：匿名粘性",
          desc: state.authMode === "gateway-user"
            ? (userModeReady ? "已选择按用户分发模式。" : "你选择了用户分发模式，还需要填入 ws-... 令牌。")
            : "每台客户端会生成独立占位 key，不会把真实客户端身份透传给上游。"
        },
        {
          tone: gatewayReady ? "ready" : "active",
          title: gatewayReady ? "Gateway 地址已填写" : "填写 Gateway 根地址",
          desc: gatewayReady ? "当前准备写入：" + currentGatewayValue() : "例如 https://gateway.example.com，不要带 /proxy。"
        },
        {
          tone: readyToApply ? "ready" : "active",
          title: readyToApply ? "可以点击一键应用 Patch" : "等待 Patch 条件满足",
          desc: readyToApply ? "应用成功后重启 Windsurf 即可开始测试。" : "只要 Gateway 地址和必要令牌填好，这一步就会变为可执行。"
        }
      ]
      renderGuide(guideItems)
    }

    async function refreshDetect(message) {
      setStatus("正在检测本机 Windsurf 环境...", false)
      try {
        const result = await request("/api/state" + currentQuery(), { method: "GET" })
        render(result)
        setStatus(message || "检测完成，请继续下一步。", false)
      } catch (error) {
        setStatus(error.message, true)
      }
    }

    async function applyPatch() {
      setStatus("正在应用 Patch，请稍候...", false)
      const payload = {
        gateway_url: $("gateway").value.trim(),
        register_gateway_url: $("register-gateway").value.trim(),
        config_dir: $("config-dir").value.trim(),
        install_dir: $("install-dir").value.trim(),
        mode: $("patch-mode").value
      }
      if (state.authMode === "gateway-user") {
        payload.auth_token = $("auth-token").value.trim()
      }
      try {
        const result = await request("/api/apply", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload)
        })
        render(result.detect)
        renderMessages([
          "备份目录：" + (result.backup_dir || "本次未生成"),
          "最终模式：" + (result.effective_auth_mode || "-")
        ].concat(result.messages || [], result.detect.messages || []))
        setStatus("Patch 已应用完成。现在重启 Windsurf，再进入客户端测试。", false)
      } catch (error) {
        setStatus(error.message, true)
      }
    }

    async function restoreBackup() {
      setStatus("正在恢复最近一次备份...", false)
      try {
        const result = await request("/api/restore", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            config_dir: $("config-dir").value.trim(),
            install_dir: $("install-dir").value.trim()
          })
        })
        render(result.detect)
        renderMessages([
          "恢复来源：" + result.restored_from
        ].concat(result.messages || [], result.detect.messages || []))
        setStatus("最近一次备份已恢复。请重启 Windsurf。", false)
      } catch (error) {
        setStatus(error.message, true)
      }
    }

    $("mode-anon").addEventListener("click", function() { setAuthMode("anonymous") })
    $("mode-user").addEventListener("click", function() { setAuthMode("gateway-user") })
    $("apply").addEventListener("click", applyPatch)
    $("refresh").addEventListener("click", function() { refreshDetect("检测完成，请确认右侧摘要后继续。") })
    $("restore").addEventListener("click", restoreBackup)
    $("gateway").addEventListener("input", refreshWizardState)
    $("auth-token").addEventListener("input", refreshWizardState)
    $("config-dir").addEventListener("input", refreshWizardState)
    $("install-dir").addEventListener("input", refreshWizardState)

    refreshWizardState()
    refreshDetect("已自动完成首次检测，请按卡片步骤继续。")
  </script>
</body>
</html>`
