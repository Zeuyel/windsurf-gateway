# Windsurf Gateway Patch Tool

This tool redirects Windsurf API traffic to a custom Windsurf Gateway. After patching, the Windsurf client should not need to log in to a real Windsurf account; backend account tokens are selected by the gateway token pool.

Each patched Windsurf install also gets a local placeholder API key. The gateway uses that key only for internal sticky routing. It is replaced by the real backend account token before forwarding upstream.

You can also pass an optional gateway user token (`ws-...`). In that mode the patched client sends the `ws-...` token only to your gateway for per-user routing and policy; the gateway still replaces it with the backend token before forwarding upstream.

## What it patches

Windsurf has a configurable API server setting:

```json
{
  "codeium.apiServerUrl": "https://your-gateway.example.com"
}
```

The default API endpoint found in the bundled extension is:

```text
https://server.codeium.com
```

The tool supports three patch targets:

- `settings.json`: user-level config override.
- `settings.json`: user-level config override, and disables the `devin-cloud` ACP connector in gateway mode.
- `state.vscdb`: Windsurf global state cache, if present.
- `extension.js`: bundled extension constant replacement plus gateway auth-session fallback and user-status fallback.
- `windsurf-gateway-patch.json`: local patch metadata used to persist the per-client placeholder key.

## Detect

```bash
node patch.js detect
```

## Patch

Recommended config-level patch:

```bash
node patch.js --gateway=https://your-gateway.example.com --mode=config
```

Full patch:

```bash
node patch.js --gateway=https://your-gateway.example.com --mode=all
```

Patch with a gateway user token:

```bash
node patch.js --gateway=https://your-gateway.example.com --auth-token=ws-xxxxxxxx --mode=all
```

Environment variables:

```bash
WINDSURF_GATEWAY_URL=https://your-gateway.example.com node patch.js --mode=config
```

```bash
WINDSURF_GATEWAY_URL=https://your-gateway.example.com \
WINDSURF_GATEWAY_USER_TOKEN=ws-xxxxxxxx \
node patch.js --mode=all
```

## Restore

```bash
node patch.js --restore
```

## Custom paths

```bash
WINDSURF_CONFIG_DIR=/path/to/Windsurf/config \
WINDSURF_INSTALL_DIR=/path/to/windsurf/resources/app \
node patch.js detect
```

## Linux paths

Default Linux config path:

```text
~/.config/Windsurf
```

Default Linux install path:

```text
/opt/windsurf/resources/app
```

## Notes

- `--mode=config` does not require root if your user owns the Windsurf config directory.
- `--mode=extension` or `--mode=all` may require elevated permissions because `/opt/windsurf` is usually root-owned.
- Close Windsurf before patching or restoring so `state.vscdb` and `extension.js` are not held open by the app.
- If Windsurf still shows `Log in to Windsurf` after gateway `Ping` already returns `200`, rerun with `--mode=all` so the bundled extension gets the auth-session fallback patch.
- If gateway `Ping` returns `200` but the top bar still flips back to `Log in to Windsurf`, rerun with `--mode=all` from the latest repo so the bundled extension also gets the user-status fallback patch.
- If you patched with an older repo version, rerun `node patch.js --gateway=... --mode=all` once so the legacy shared placeholder key is upgraded to a per-client placeholder key.
- In gateway mode the patch also writes `windsurf.acp.enabledAgents.devin-cloud = false` to suppress the remote Devin Cloud ACP connector, which otherwise may show `Devin Cloud is disconnected.` while core model RPCs are unrelated.
- `node patch.js detect` now shows whether both extension fallbacks are already present.
- Restart Windsurf after patching.
- Point `codeium.apiServerUrl` to the gateway root URL, not to `/proxy`; the gateway catches native Windsurf API paths and forwards them with backend pool tokens.
