# Windsurf Gateway Patch Tool

This tool redirects Windsurf API traffic to a custom Windsurf Gateway.

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

The default register endpoint is:

```text
https://register.windsurf.com
```

The tool supports three patch targets:

- `settings.json`: user-level config override.
- `state.vscdb`: Windsurf global state cache, if present.
- `extension.js`: bundled extension constant replacement.

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

With register gateway override:

```bash
node patch.js --gateway=https://your-gateway.example.com --register-gateway=https://your-gateway.example.com --mode=all
```

Environment variables:

```bash
WINDSURF_GATEWAY_URL=https://your-gateway.example.com node patch.js --mode=config
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
- Restart Windsurf after patching.
