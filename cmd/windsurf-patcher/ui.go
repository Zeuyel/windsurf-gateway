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
        radial-gradient(circle at 12% 18%, rgba(19, 115, 104, .18), transparent 30%),
        radial-gradient(circle at 88% 10%, rgba(198, 96, 47, .16), transparent 26%),
        linear-gradient(145deg, #f3eee6 0%, #edf5ef 46%, #eef4fa 100%);
      --panel: rgba(255, 255, 255, .9);
      --panel-strong: rgba(255, 255, 255, .96);
      --ink: #18263b;
      --muted: #64748b;
      --line: rgba(24, 38, 59, .11);
      --accent: #137368;
      --accent-soft: rgba(19, 115, 104, .1);
      --warn: #c6602f;
      --warn-soft: rgba(198, 96, 47, .1);
      --ok: #1f8b4c;
      --ok-soft: rgba(31, 139, 76, .1);
      --shadow: 0 24px 64px rgba(24, 38, 59, .12);
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
    button, input, select { font: inherit; }
    .page {
      width: min(1240px, calc(100vw - 32px));
      margin: 24px auto;
      display: grid;
      grid-template-columns: minmax(0, 1.08fr) minmax(320px, .92fr);
      gap: 22px;
    }
    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: var(--radius-lg);
      box-shadow: var(--shadow);
      backdrop-filter: blur(18px);
    }
    .wizard {
      padding: 28px;
      display: grid;
      gap: 18px;
    }
    .hero {
      padding: 26px;
      border-radius: 24px;
      background:
        linear-gradient(135deg, rgba(19, 115, 104, .93), rgba(15, 59, 87, .92)),
        linear-gradient(135deg, rgba(255,255,255,.16), rgba(255,255,255,0));
      color: #fff;
      position: relative;
      overflow: hidden;
    }
    .hero::after {
      content: "";
      position: absolute;
      right: -34px;
      bottom: -44px;
      width: 210px;
      height: 210px;
      border-radius: 999px;
      background: rgba(255,255,255,.08);
    }
    .hero h1 {
      margin: 0;
      font-size: clamp(30px, 4vw, 40px);
      line-height: 1.02;
      letter-spacing: -.04em;
      position: relative;
      z-index: 1;
    }
    .hero p {
      margin: 12px 0 0;
      max-width: 720px;
      line-height: 1.72;
      color: rgba(255,255,255,.86);
      position: relative;
      z-index: 1;
    }
    .hero-tags {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      margin-top: 16px;
      position: relative;
      z-index: 1;
    }
    .hero-tag {
      padding: 8px 12px;
      border-radius: 999px;
      background: rgba(255,255,255,.12);
      border: 1px solid rgba(255,255,255,.16);
      font-size: 13px;
      color: rgba(255,255,255,.92);
    }
    .notice {
      padding: 16px 18px;
      border-radius: 18px;
      background: var(--warn-soft);
      border: 1px solid rgba(198, 96, 47, .16);
      line-height: 1.7;
      color: #81492d;
    }
    .progress-bar {
      display: grid;
      grid-template-columns: repeat(5, minmax(0, 1fr));
      gap: 10px;
    }
    .progress-chip {
      padding: 12px;
      border-radius: 18px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,.78);
      text-align: left;
      cursor: pointer;
      transition: transform .16s ease, border-color .16s ease, box-shadow .16s ease;
    }
    .progress-chip:hover {
      transform: translateY(-1px);
      box-shadow: 0 10px 24px rgba(24, 38, 59, .08);
    }
    .progress-chip.active {
      border-color: rgba(19, 115, 104, .34);
      background: linear-gradient(180deg, rgba(19, 115, 104, .12), rgba(19, 115, 104, .04));
    }
    .progress-chip.ready {
      border-color: rgba(31, 139, 76, .2);
    }
    .progress-chip span {
      display: block;
      font-size: 12px;
      color: var(--muted);
      margin-bottom: 6px;
    }
    .progress-chip strong {
      display: block;
      font-size: 14px;
      color: var(--ink);
      line-height: 1.45;
    }
    .stage {
      display: grid;
      gap: 16px;
    }
    .step-card {
      padding: 24px;
      border-radius: 24px;
      border: 1px solid var(--line);
      background: var(--panel-strong);
      display: grid;
      gap: 16px;
    }
    .step-card.hidden { display: none; }
    .step-head {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 16px;
    }
    .step-title {
      display: flex;
      gap: 14px;
      align-items: flex-start;
    }
    .step-index {
      width: 40px;
      height: 40px;
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
      font-size: 22px;
      letter-spacing: -.02em;
    }
    .step-head p {
      margin: 6px 0 0;
      color: var(--muted);
      line-height: 1.72;
      font-size: 14px;
    }
    .pill {
      flex: none;
      padding: 9px 12px;
      border-radius: 999px;
      font-size: 12px;
      font-weight: 700;
      background: rgba(100, 116, 139, .08);
      color: var(--muted);
      border: 1px solid var(--line);
    }
    .pill.ready {
      background: var(--ok-soft);
      color: var(--ok);
      border-color: rgba(31, 139, 76, .18);
    }
    .pill.active {
      background: var(--accent-soft);
      color: var(--accent);
      border-color: rgba(19, 115, 104, .18);
    }
    .pill.warn {
      background: var(--warn-soft);
      color: var(--warn);
      border-color: rgba(198, 96, 47, .16);
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
      border-color: rgba(19, 115, 104, .4);
      box-shadow: 0 0 0 4px rgba(19, 115, 104, .09);
    }
    .field-hint {
      margin-top: 8px;
      color: var(--muted);
      line-height: 1.6;
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
      text-align: left;
      cursor: pointer;
      transition: transform .16s ease, border-color .16s ease, box-shadow .16s ease, background .16s ease;
    }
    .mode-card:hover {
      transform: translateY(-1px);
      box-shadow: 0 12px 28px rgba(24, 38, 59, .08);
    }
    .mode-card.active {
      border-color: rgba(19, 115, 104, .34);
      background: linear-gradient(180deg, rgba(19, 115, 104, .12), rgba(19, 115, 104, .04));
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
    .mode-note {
      margin-top: 10px;
      display: inline-flex;
      padding: 6px 10px;
      border-radius: 999px;
      background: rgba(24, 38, 59, .06);
      color: var(--muted);
      font-size: 12px;
      font-weight: 700;
    }
    .soft-box {
      padding: 14px 16px;
      border-radius: 16px;
      background: rgba(24, 38, 59, .04);
      color: var(--muted);
      line-height: 1.7;
      font-size: 14px;
    }
    .path-box {
      display: grid;
      gap: 10px;
    }
    .path-item {
      padding: 12px 14px;
      border-radius: 16px;
      background: rgba(24, 38, 59, .04);
      border: 1px solid rgba(24, 38, 59, .05);
    }
    .path-item strong {
      display: block;
      font-size: 12px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: .06em;
      margin-bottom: 6px;
    }
    .path-item span,
    .path-item code {
      color: var(--ink);
      line-height: 1.55;
      word-break: break-all;
    }
    .action-row {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
    }
    .action-row button,
    .nav-row button,
    .detail-toggle {
      border: 0;
      border-radius: 16px;
      padding: 13px 18px;
      font-weight: 700;
      cursor: pointer;
      transition: transform .16s ease, filter .16s ease, opacity .16s ease;
    }
    .action-row button:hover,
    .nav-row button:hover,
    .detail-toggle:hover {
      transform: translateY(-1px);
      filter: brightness(.98);
    }
    .btn-primary {
      background: var(--accent);
      color: #fff;
    }
    .btn-secondary {
      background: rgba(24, 38, 59, .08);
      color: var(--ink);
    }
    .btn-danger {
      background: var(--warn);
      color: #fff;
    }
    .btn-ghost {
      background: rgba(24, 38, 59, .05);
      color: var(--ink);
      border: 1px solid rgba(24, 38, 59, .08);
    }
    button:disabled {
      cursor: not-allowed;
      opacity: .5;
      transform: none !important;
      filter: none !important;
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
    .status-card.error .status-text { color: #a63c1f; }
    .status-label {
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .08em;
      color: var(--muted);
    }
    .status-text {
      font-size: 15px;
      line-height: 1.65;
      color: var(--ink);
    }
    .nav-row {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      padding-top: 4px;
    }
    .nav-hint {
      color: var(--muted);
      font-size: 14px;
      line-height: 1.6;
    }
    .board {
      padding: 28px;
      display: grid;
      gap: 16px;
      align-content: start;
    }
    .board h3 {
      margin: 0;
      font-size: 24px;
      letter-spacing: -.03em;
    }
    .board p {
      margin: 8px 0 0;
      color: var(--muted);
      line-height: 1.7;
      font-size: 14px;
    }
    .summary-grid {
      display: grid;
      gap: 12px;
    }
    .summary-card,
    .detail-card {
      padding: 20px;
      border-radius: 22px;
      border: 1px solid var(--line);
      background: var(--panel-strong);
    }
    .summary-card strong,
    .detail-card h4 {
      display: block;
      margin: 0 0 10px;
      font-size: 17px;
      color: var(--ink);
    }
    .summary-value {
      font-size: 24px;
      font-weight: 800;
      line-height: 1.2;
      letter-spacing: -.03em;
      word-break: break-word;
    }
    .summary-meta {
      margin-top: 8px;
      color: var(--muted);
      line-height: 1.65;
      font-size: 14px;
    }
    .summary-mini {
      display: grid;
      gap: 10px;
      margin-top: 4px;
    }
    .summary-mini-item {
      padding-top: 10px;
      border-top: 1px solid rgba(24, 38, 59, .08);
    }
    .summary-mini-item:first-child { border-top: 0; padding-top: 0; }
    .summary-mini-item span {
      display: block;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .06em;
      color: var(--muted);
      margin-bottom: 6px;
    }
    .summary-mini-item strong {
      margin: 0;
      font-size: 16px;
      line-height: 1.5;
      word-break: break-word;
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
      gap: 10px;
      align-items: flex-start;
      padding: 10px 0;
      border-top: 1px solid rgba(24, 38, 59, .08);
    }
    .check-item:first-child { border-top: 0; padding-top: 0; }
    .check-mark {
      width: 24px;
      height: 24px;
      border-radius: 999px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      font-size: 12px;
      font-weight: 700;
      background: rgba(24, 38, 59, .08);
      color: var(--muted);
      flex: none;
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
      margin: 0 0 4px;
      font-size: 14px;
    }
    .check-copy span {
      display: block;
      color: var(--muted);
      font-size: 13px;
      line-height: 1.6;
    }
    .detail-toggle {
      width: 100%;
      background: rgba(24, 38, 59, .08);
      color: var(--ink);
    }
    .detail-stack {
      display: grid;
      gap: 12px;
    }
    .detail-card.hidden { display: none; }
    .detail-grid {
      display: grid;
      gap: 10px;
    }
    .detail-row {
      display: grid;
      grid-template-columns: 140px 1fr;
      gap: 12px;
      align-items: start;
      padding-top: 10px;
      border-top: 1px solid rgba(24, 38, 59, .08);
    }
    .detail-row:first-child { border-top: 0; padding-top: 0; }
    .detail-row strong {
      color: var(--muted);
      font-size: 13px;
    }
    .detail-row span,
    .detail-row code {
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
    .hidden { display: none; }
    @media (max-width: 1080px) {
      .page { grid-template-columns: 1fr; }
      .progress-bar { grid-template-columns: repeat(2, minmax(0, 1fr)); }
    }
    @media (max-width: 760px) {
      .wizard, .board { padding: 18px; }
      .hero, .step-card, .summary-card, .detail-card { padding: 18px; }
      .field-row, .mode-grid, .progress-bar { grid-template-columns: 1fr; }
      .nav-row { flex-direction: column; align-items: stretch; }
      .detail-row { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="page">
    <section class="panel wizard">
      <div class="hero">
        <h1>Windsurf Patch 向导</h1>
        <p>这个页面只做一件事：把 Windsurf 按步骤接到你的 Gateway。界面一次只显示当前一步，用户不需要同时理解所有技术细节。</p>
        <div class="hero-tags">
          <span class="hero-tag">本地页面，不上传配置</span>
          <span class="hero-tag">支持匿名粘性 / 用户令牌</span>
          <span class="hero-tag">自动备份，可一键恢复</span>
        </div>
      </div>

      <div class="notice">
        开始前先完全退出 Windsurf。这样 <code>state.vscdb</code> 和内置客户端脚本 <code>extension.js</code> 不会被占用，应用或回滚 patch 时更稳定。
      </div>

      <div class="progress-bar" id="progress-bar"></div>

      <div class="stage">
        <section class="step-card" data-step-card="1">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">1</div>
              <div>
                <h2>检测本机 Windsurf</h2>
                <p>先确认本机配置目录、安装目录和当前 patch 状态。检测完成后，再进入下一步。</p>
              </div>
            </div>
            <span class="pill" id="pill-detect">待检测</span>
          </div>
          <div class="action-row">
            <button class="btn-secondary" id="refresh" type="button">重新检测</button>
          </div>
          <div class="path-box">
            <div class="path-item">
              <strong>配置目录</strong>
              <span id="path-config">-</span>
            </div>
            <div class="path-item">
              <strong>安装目录</strong>
              <span id="path-install">-</span>
            </div>
          </div>
          <div class="soft-box">如果你的检测结果已经正确，就不需要提前手动填写路径。后面的高级路径步骤可以直接跳过。</div>
        </section>

        <section class="step-card hidden" data-step-card="2">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">2</div>
              <div>
                <h2>选择接入模式</h2>
                <p>这里决定客户端如何接入 Gateway。匿名粘性更轻，网关用户令牌更适合多人共享并做用户级管控。</p>
              </div>
            </div>
            <span class="pill active" id="pill-mode">匿名粘性</span>
          </div>
          <div class="mode-grid">
            <button id="mode-anon" class="mode-card active" type="button">
              <strong>匿名粘性</strong>
              <span>每台客户端生成独立占位 key，只进入 Gateway，不把真实客户端身份透传给上游。</span>
              <span class="mode-note">适合 Gateway 允许匿名接入时使用</span>
            </button>
            <button id="mode-user" class="mode-card" type="button">
              <strong>网关用户令牌</strong>
              <span>客户端先携带后台分配的 <code>ws-...</code> 用户令牌接入，再由 Gateway 按用户做限速、封禁和策略分发。</span>
              <span class="mode-note">适合管理员开启用户鉴权时使用</span>
            </button>
          </div>
          <div id="auth-block" class="field hidden">
            <label for="auth-token">Gateway 用户令牌</label>
            <input id="auth-token" placeholder="ws-xxxxxxxx">
            <div class="field-hint">这个令牌只会发送给你的 Gateway，不会上送到 Windsurf 上游。</div>
          </div>
        </section>

        <section class="step-card hidden" data-step-card="3">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">3</div>
              <div>
                <h2>填写 Gateway 地址</h2>
                <p>填 Gateway 根地址，不要带 <code>/proxy</code>。如果你的 Gateway 也代理 Windsurf 登录/注册接口，再额外填写下面的登录/注册地址。</p>
              </div>
            </div>
            <span class="pill" id="pill-gateway">待填写</span>
          </div>
          <div class="field-grid">
            <div class="field">
              <label for="gateway">Gateway 地址</label>
              <input id="gateway" placeholder="https://gateway.example.com">
              <div class="field-hint">最终会写入 <code>codeium.apiServerUrl</code>。</div>
            </div>
            <div class="field-row">
              <div class="field">
                <label for="register-gateway">登录/注册接口地址</label>
                <input id="register-gateway" placeholder="可选">
                <div class="field-hint">只在你的 Gateway 也接管 <code>codeium.registerApiServerUrl</code> 时填写；普通代理场景留空即可。</div>
              </div>
              <div class="field">
                <label for="patch-mode">改写范围</label>
                <select id="patch-mode">
                  <option value="all">完整改写：配置 + 全局状态 + 客户端脚本</option>
                  <option value="config">仅配置和全局状态</option>
                  <option value="extension">仅客户端脚本</option>
                </select>
                <div class="field-hint">第一次接入建议使用完整 patch。</div>
              </div>
            </div>
          </div>
        </section>

        <section class="step-card hidden" data-step-card="4">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">4</div>
              <div>
                <h2>高级路径设置</h2>
                <p>只有在便携版、自定义安装目录或多套配置共存时，才需要手动填写。大多数人可以直接下一步。</p>
              </div>
            </div>
            <span class="pill" id="pill-advanced">默认自动识别</span>
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
          <div class="soft-box">如果这里填写了新路径，建议回到第一步重新检测一次，确认路径解析结果没有偏。</div>
        </section>

        <section class="step-card hidden" data-step-card="5">
          <div class="step-head">
            <div class="step-title">
              <div class="step-index">5</div>
              <div>
                <h2>执行 Patch</h2>
                <p>确认完前面四步后，直接应用。需要回退时，可以恢复最近一次自动备份。</p>
              </div>
            </div>
            <span class="pill" id="pill-run">等待前置步骤</span>
          </div>
          <div class="action-row">
            <button class="btn-primary" id="apply" type="button">一键应用 Patch</button>
            <button class="btn-danger" id="restore" type="button">恢复最近一次备份</button>
          </div>
          <div class="status-card" id="status-card">
            <div class="status-label">当前状态</div>
            <div class="status-text" id="status-text">正在准备检测本机环境。</div>
          </div>
          <button class="detail-toggle" id="toggle-details" type="button">显示开发详情</button>
        </section>
      </div>

      <div class="nav-row">
        <button class="btn-ghost" id="prev-step" type="button">上一步</button>
        <div class="nav-hint" id="nav-hint">先完成本机检测，确认路径正常。</div>
        <button class="btn-secondary" id="next-step" type="button">下一步</button>
      </div>
    </section>

    <section class="panel board">
      <div>
        <h3>当前选择</h3>
        <p>这里只保留用户真正需要确认的摘要。开发用路径、patch 健康状态和执行消息默认折叠，避免增加心智负担。</p>
      </div>

      <div class="summary-grid">
        <div class="summary-card">
          <strong>当前 Gateway</strong>
          <div class="summary-value" id="summary-gateway">-</div>
          <div class="summary-meta" id="summary-gateway-meta">还没有填写地址。</div>
        </div>
        <div class="summary-card">
          <strong>接入模式</strong>
          <div class="summary-value" id="summary-auth">匿名粘性</div>
          <div class="summary-meta" id="summary-auth-meta">默认不会把真实客户端身份直接透传给上游。</div>
        </div>
        <div class="summary-card">
          <strong>本次改写范围</strong>
          <div class="summary-value" id="summary-scope">完整改写</div>
          <div class="summary-meta" id="summary-scope-meta">首次接入通常推荐完整 patch。</div>
        </div>
        <div class="summary-card">
          <strong>下一步提醒</strong>
          <div class="summary-mini">
            <div class="summary-mini-item">
              <span>当前步骤</span>
              <strong id="summary-step">1 / 5</strong>
            </div>
            <div class="summary-mini-item">
              <span>Patch 状态</span>
              <strong id="summary-patch">未检测</strong>
            </div>
          </div>
        </div>
      </div>

      <div class="detail-card">
        <h4>当前进度</h4>
        <ul class="checklist" id="guide-list"></ul>
      </div>

      <div class="detail-stack" id="detail-stack">
        <div class="detail-card hidden" id="detail-paths">
          <h4>路径与文件</h4>
          <div class="detail-grid">
            <div class="detail-row"><strong>settings.json</strong><code id="path-settings">-</code></div>
            <div class="detail-row"><strong>state.vscdb</strong><code id="path-state">-</code></div>
            <div class="detail-row"><strong>客户端脚本</strong><code id="path-extension">-</code></div>
          </div>
        </div>

        <div class="detail-card hidden" id="detail-health">
          <h4>Patch 健康状态</h4>
          <div class="detail-grid">
            <div class="detail-row"><strong>配置状态</strong><span id="health-settings">-</span></div>
            <div class="detail-row"><strong>全局状态</strong><span id="health-global">-</span></div>
            <div class="detail-row"><strong>客户端脚本兜底</strong><span id="health-extension">-</span></div>
            <div class="detail-row"><strong>本地占位 Key</strong><span id="health-placeholder">-</span></div>
          </div>
        </div>

        <div class="detail-card hidden" id="detail-messages">
          <h4>执行消息</h4>
          <ul class="message-list" id="messages"></ul>
        </div>
      </div>
    </section>
  </div>

  <script>
    const state = {
      authMode: 'anonymous',
      detect: null,
      currentStep: 1,
      developerOpen: false,
      totalSteps: 5
    }

    const stepTitles = [
      '检测环境',
      '选择模式',
      '填写地址',
      '高级路径',
      '执行 Patch'
    ]

    const $ = function(id) { return document.getElementById(id) }
    const statusCard = $('status-card')
    const statusText = $('status-text')
    const guideList = $('guide-list')
    const progressBar = $('progress-bar')
    const detailIds = ['detail-paths', 'detail-health', 'detail-messages']

    function setStatus(text, isError) {
      statusText.textContent = text
      statusCard.classList.toggle('error', !!isError)
    }

    function setPill(id, text, tone) {
      const el = $(id)
      el.textContent = text
      el.className = 'pill'
      if (tone) {
        el.classList.add(tone)
      }
    }

    function currentGatewayValue() {
      return $('gateway').value.trim()
    }

    function currentAuthTokenValue() {
      return $('auth-token').value.trim()
    }

    function currentPatchModeValue() {
      return $('patch-mode').value
    }

    function currentPatchModeLabel() {
      switch (currentPatchModeValue()) {
        case 'config':
          return '仅配置与全局状态'
        case 'extension':
          return '仅客户端脚本'
        default:
          return '完整改写'
      }
    }

    function hasDetectResult() {
      return !!(state.detect && state.detect.environment)
    }

    function userModeReady() {
      return state.authMode !== 'gateway-user' || !!currentAuthTokenValue()
    }

    function gatewayReady() {
      return !!currentGatewayValue()
    }

    function advancedCustomized() {
      return !!$('config-dir').value.trim() || !!$('install-dir').value.trim()
    }

    function readyToApply() {
      return hasDetectResult() && userModeReady() && gatewayReady()
    }

    function canAdvance(step) {
      switch (step) {
        case 1:
          return hasDetectResult()
        case 2:
          return userModeReady()
        case 3:
          return gatewayReady()
        case 4:
          return true
        default:
          return false
      }
    }

    function textOrDash(value) {
      return value && String(value).trim() ? String(value) : '-'
    }

    function toggleDeveloperDetails(forceOpen) {
      if (typeof forceOpen === 'boolean') {
        state.developerOpen = forceOpen
      } else {
        state.developerOpen = !state.developerOpen
      }
      detailIds.forEach(function(id) {
        $(id).classList.toggle('hidden', !state.developerOpen)
      })
      $('toggle-details').textContent = state.developerOpen ? '隐藏开发详情' : '显示开发详情'
    }

    function setAuthMode(mode) {
      state.authMode = mode
      $('mode-anon').classList.toggle('active', mode === 'anonymous')
      $('mode-user').classList.toggle('active', mode === 'gateway-user')
      $('auth-block').classList.toggle('hidden', mode !== 'gateway-user')
      refreshWizardState()
    }

    async function request(url, options) {
      const response = await fetch(url, options)
      const payload = await response.json()
      if (!payload.ok) {
        throw new Error(payload.error || '请求失败')
      }
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
      const list = $('messages')
      list.innerHTML = ''
      ;(items || []).forEach(function(item) {
        const li = document.createElement('li')
        li.textContent = item
        list.appendChild(li)
      })
      if (!list.children.length) {
        const li = document.createElement('li')
        li.textContent = '暂无额外消息。'
        list.appendChild(li)
      }
    }

    function displayAuthMode(result) {
      const mode = (result.global_state && result.global_state.auth_mode) || (result.extension && result.extension.auth_mode) || ''
      switch (mode) {
        case 'gateway-user':
          return '网关用户令牌'
        case 'per-client-placeholder':
          return '匿名粘性'
        case 'legacy-shared-placeholder':
          return '旧版共享占位'
        case 'custom':
          return '自定义令牌'
        case 'none':
          return '未注入'
        default:
          return mode || '未检测'
      }
    }

    function displayPatchState(result) {
      switch (result.patch_state_mode) {
        case 'gateway-user':
          return '用户分发'
        case 'per-client-placeholder':
          return '匿名粘性'
        case 'legacy-shared-placeholder':
          return '旧版占位'
        case 'none':
        case '':
          return result.patch_state_exists ? '已存在但未识别' : '未创建'
        default:
          return result.patch_state_mode || '未创建'
      }
    }

    function describeSettings(result) {
      if (!result.settings.exists) return '未找到 settings.json'
      if (result.settings.gateway_url && result.settings.devin_cloud_disabled) return '已写入 Gateway，且已关闭 devin-cloud'
      if (result.settings.gateway_url) return '已写入 Gateway'
      return '已找到 settings.json，但尚未写入 Gateway'
    }

    function describeGlobal(result) {
      if (!result.global_state.exists) return result.global_state.read_error || '未找到 state.vscdb'
      if (result.global_state.auth_token_summary) return '已注入认证状态：' + result.global_state.auth_token_summary
      if (result.global_state.onboarding_patched || result.global_state.education_patched) return '已写入 onboarding/教育状态'
      return '已找到 state.vscdb，但未看到认证兜底'
    }

    function describeExtension(result) {
      if (!result.extension.exists) return result.extension.read_error || '未找到内置客户端脚本 extension.js'
      if (result.extension.has_auth_fallback && result.extension.has_user_status_fallback) return '登录兜底和用户状态兜底都已存在'
      if (result.extension.has_auth_fallback) return '仅存在登录兜底，还未写入用户状态兜底'
      return '尚未注入客户端脚本兜底'
    }

    function render(result) {
      state.detect = result

      $('path-config').textContent = textOrDash(result.environment.config_dir)
      $('path-install').textContent = textOrDash(result.environment.install_dir)
      $('path-settings').textContent = textOrDash(result.environment.settings_path)
      $('path-state').textContent = textOrDash(result.environment.global_state_path)
      $('path-extension').textContent = textOrDash(result.environment.extension_path)

      $('health-settings').textContent = describeSettings(result)
      $('health-global').textContent = describeGlobal(result)
      $('health-extension').textContent = describeExtension(result)
      $('health-placeholder').textContent = result.patch_state_placeholder || '未生成'
      $('summary-patch').textContent = displayPatchState(result)

      if (!$('gateway').value && result.settings.gateway_url) {
        $('gateway').value = result.settings.gateway_url
      }
      if (!$('register-gateway').value && result.settings.register_gateway_url) {
        $('register-gateway').value = result.settings.register_gateway_url
      }
      if (result.global_state.auth_mode === 'gateway-user' || result.extension.auth_mode === 'gateway-user') {
        setAuthMode('gateway-user')
      }
      renderMessages(result.messages)
      refreshWizardState()
    }

    function renderProgress(items) {
      progressBar.innerHTML = ''
      items.forEach(function(item, index) {
        const btn = document.createElement('button')
        btn.type = 'button'
        btn.className = 'progress-chip'
        if (item.tone) btn.classList.add(item.tone)
        if (state.currentStep === index + 1) btn.classList.add('active')
        btn.innerHTML = '<span>步骤 ' + (index + 1) + '</span><strong>' + item.title + '</strong>'
        btn.addEventListener('click', function() {
          goToStep(index + 1)
        })
        progressBar.appendChild(btn)
      })
    }

    function renderGuide(items) {
      guideList.innerHTML = ''
      items.forEach(function(item) {
        const li = document.createElement('li')
        li.className = 'check-item ' + item.tone

        const mark = document.createElement('span')
        mark.className = 'check-mark'
        mark.textContent = item.tone === 'ready' ? '✓' : item.tone === 'active' ? '→' : '·'

        const copy = document.createElement('div')
        copy.className = 'check-copy'

        const title = document.createElement('strong')
        title.textContent = item.title
        const desc = document.createElement('span')
        desc.textContent = item.desc

        copy.appendChild(title)
        copy.appendChild(desc)
        li.appendChild(mark)
        li.appendChild(copy)
        guideList.appendChild(li)
      })
    }

    function currentStepHint() {
      switch (state.currentStep) {
        case 1:
          return hasDetectResult() ? '检测完成后，点下一步进入接入模式选择。' : '先完成本机检测，确认路径正常。'
        case 2:
          return userModeReady() ? '模式已经选好，可以继续填写 Gateway 地址。' : '如果选择了用户令牌模式，需要先填入 ws 令牌。'
        case 3:
          return gatewayReady() ? '地址已具备，可以继续检查高级路径。' : '这里必须先填 Gateway 根地址。'
        case 4:
          return '如果不需要自定义路径，可以直接进入执行步骤。'
        default:
          return readyToApply() ? '现在可以直接应用 patch 或恢复最近一次备份。' : '还有前置步骤未完成，先回去补齐。'
      }
    }

    function goToStep(step) {
      if (step < 1) step = 1
      if (step > state.totalSteps) step = state.totalSteps
      state.currentStep = step
      updateStepUI()
    }

    function updateStepUI() {
      document.querySelectorAll('[data-step-card]').forEach(function(node) {
        const step = Number(node.getAttribute('data-step-card'))
        node.classList.toggle('hidden', step !== state.currentStep)
      })
      $('prev-step').disabled = state.currentStep === 1
      $('next-step').disabled = state.currentStep >= state.totalSteps || !canAdvance(state.currentStep)
      $('next-step').textContent = state.currentStep === state.totalSteps ? '已到最后一步' : (state.currentStep === state.totalSteps - 1 ? '去执行' : '下一步')
      $('summary-step').textContent = state.currentStep + ' / ' + state.totalSteps
      $('nav-hint').textContent = currentStepHint()
      document.title = 'Windsurf Patch 向导 - ' + stepTitles[state.currentStep - 1]
    }

    function refreshWizardState() {
      const detectReady = hasDetectResult()
      const modeReady = userModeReady()
      const gatewayDone = gatewayReady()
      const advancedDone = advancedCustomized()
      const applyReady = readyToApply()

      setPill('pill-detect', detectReady ? '已检测' : '待检测', detectReady ? 'ready' : 'warn')
      setPill('pill-mode', state.authMode === 'gateway-user' ? (modeReady ? '用户令牌模式' : '等待令牌') : '匿名粘性', state.authMode === 'gateway-user' ? (modeReady ? 'ready' : 'warn') : 'active')
      setPill('pill-gateway', gatewayDone ? '地址已填写' : '待填写', gatewayDone ? 'ready' : 'warn')
      setPill('pill-advanced', advancedDone ? '使用自定义路径' : '默认自动识别', advancedDone ? 'active' : '')
      setPill('pill-run', applyReady ? '可以执行' : '等待前置步骤', applyReady ? 'ready' : 'warn')

      $('summary-gateway').textContent = gatewayDone ? currentGatewayValue() : '-'
      $('summary-gateway-meta').textContent = gatewayDone ? '将写入 settings.json 和需要的本地状态。' : '还没有填写地址。'
      $('summary-auth').textContent = state.authMode === 'gateway-user' ? '网关用户令牌' : '匿名粘性'
      $('summary-auth-meta').textContent = state.authMode === 'gateway-user'
        ? (modeReady ? '会使用你填写的 ws 用户令牌接入 Gateway。' : '当前模式需要先补一个 ws 用户令牌。')
        : '默认不会把真实客户端身份直接透传给上游。'
      $('summary-scope').textContent = currentPatchModeLabel()
      $('summary-scope-meta').textContent = currentPatchModeValue() === 'all'
        ? '配置、全局状态和客户端脚本都会一起改写。'
        : (currentPatchModeValue() === 'config' ? '只改配置与全局状态。' : '只改客户端脚本。')

      const progressItems = [
        { title: '检测环境', tone: detectReady ? 'ready' : 'active' },
        { title: '选择模式', tone: modeReady ? 'ready' : 'active' },
        { title: '填写地址', tone: gatewayDone ? 'ready' : 'active' },
        { title: '高级路径', tone: advancedDone ? 'ready' : '' },
        { title: '执行 Patch', tone: applyReady ? 'ready' : '' }
      ]
      renderProgress(progressItems)

      const guideItems = [
        {
          tone: detectReady ? 'ready' : (state.currentStep === 1 ? 'active' : ''),
          title: detectReady ? '本机 Windsurf 已识别' : '先完成本机检测',
          desc: detectReady ? '路径和当前 patch 状态已经拿到，可以继续。' : '点击重新检测，先确认配置目录和安装目录。'
        },
        {
          tone: modeReady ? 'ready' : (state.currentStep === 2 ? 'active' : ''),
          title: state.authMode === 'gateway-user' ? '当前模式：网关用户令牌' : '当前模式：匿名粘性',
          desc: state.authMode === 'gateway-user'
            ? (modeReady ? '用户令牌已经具备，可以按用户分发。' : '你选了用户令牌模式，还需要填一个 ws 令牌。')
            : '每台客户端会生成独立占位 key，不直接透传真实身份。'
        },
        {
          tone: gatewayDone ? 'ready' : (state.currentStep === 3 ? 'active' : ''),
          title: gatewayDone ? 'Gateway 地址已填写' : '填写 Gateway 根地址',
          desc: gatewayDone ? '准备写入：' + currentGatewayValue() : '例如 https://gateway.example.com，不要带 /proxy。'
        },
        {
          tone: applyReady ? 'ready' : (state.currentStep === 5 ? 'active' : ''),
          title: applyReady ? '可以执行 Patch' : '等待执行条件满足',
          desc: applyReady ? '应用完成后重启 Windsurf，再进行真实请求测试。' : '至少需要完成检测、模式和 Gateway 地址填写。'
        }
      ]
      renderGuide(guideItems)

      updateStepUI()
    }

    async function refreshDetect(message) {
      setStatus('正在检测本机 Windsurf 环境...', false)
      try {
        const result = await request('/api/state' + currentQuery(), { method: 'GET' })
        render(result)
        setStatus(message || '检测完成，请继续下一步。', false)
      } catch (error) {
        setStatus(error.message, true)
      }
    }

    async function applyPatch() {
      setStatus('正在应用 Patch，请稍候...', false)
      const payload = {
        gateway_url: $('gateway').value.trim(),
        register_gateway_url: $('register-gateway').value.trim(),
        config_dir: $('config-dir').value.trim(),
        install_dir: $('install-dir').value.trim(),
        mode: $('patch-mode').value
      }
      if (state.authMode === 'gateway-user') {
        payload.auth_token = $('auth-token').value.trim()
      }
      try {
        const result = await request('/api/apply', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        })
        render(result.detect)
        renderMessages([
          '备份目录：' + (result.backup_dir || '本次未生成'),
          '最终模式：' + (result.effective_auth_mode || '-')
        ].concat(result.messages || [], result.detect.messages || []))
        goToStep(5)
        setStatus('Patch 已应用完成。现在重启 Windsurf，再进入客户端测试。', false)
      } catch (error) {
        setStatus(error.message, true)
      }
    }

    async function restoreBackup() {
      setStatus('正在恢复最近一次备份...', false)
      try {
        const result = await request('/api/restore', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            config_dir: $('config-dir').value.trim(),
            install_dir: $('install-dir').value.trim()
          })
        })
        render(result.detect)
        renderMessages(['恢复来源：' + result.restored_from].concat(result.messages || [], result.detect.messages || []))
        goToStep(5)
        setStatus('最近一次备份已恢复。请重启 Windsurf。', false)
      } catch (error) {
        setStatus(error.message, true)
      }
    }

    $('mode-anon').addEventListener('click', function() { setAuthMode('anonymous') })
    $('mode-user').addEventListener('click', function() { setAuthMode('gateway-user') })
    $('apply').addEventListener('click', applyPatch)
    $('refresh').addEventListener('click', function() { refreshDetect('检测完成，请进入下一步。') })
    $('restore').addEventListener('click', restoreBackup)
    $('toggle-details').addEventListener('click', function() { toggleDeveloperDetails() })
    $('prev-step').addEventListener('click', function() { goToStep(state.currentStep - 1) })
    $('next-step').addEventListener('click', function() {
      if (state.currentStep < state.totalSteps && canAdvance(state.currentStep)) {
        goToStep(state.currentStep + 1)
      }
    })
    $('gateway').addEventListener('input', refreshWizardState)
    $('register-gateway').addEventListener('input', refreshWizardState)
    $('auth-token').addEventListener('input', refreshWizardState)
    $('config-dir').addEventListener('input', refreshWizardState)
    $('install-dir').addEventListener('input', refreshWizardState)
    $('patch-mode').addEventListener('change', refreshWizardState)

    toggleDeveloperDetails(false)
    refreshWizardState()
    refreshDetect('已自动完成首次检测，请按步骤继续。')
  </script>
</body>
</html>`
