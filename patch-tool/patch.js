#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const os = require('os');

const WINDSURF_CONFIG_DIR = path.join(os.homedir(), '.config', 'Windsurf');
const SETTINGS_PATH = path.join(WINDSURF_CONFIG_DIR, 'User', 'settings.json');
const BACKUP_DIR = path.join(WINDSURF_CONFIG_DIR, 'manual-auth-restore-' + Date.now().toString(36));

const DEFAULT_API_URL = 'https://server.codeium.com';
const CONFIG_KEY = 'codeium.apiServerUrl';

function printBanner() {
    console.log(`
╔══════════════════════════════════════════════╗
║       Windsurf Open Patch Tool v1.0.0        ║
║   Redirect Windsurf API to custom gateway     ║
╚══════════════════════════════════════════════╝
`);
}

function getGatewayUrl() {
    const envUrl = process.env.WINDSURF_GATEWAY_URL;
    if (envUrl) return envUrl;

    const args = process.argv.slice(2);
    const urlArg = args.find(a => a.startsWith('--gateway='));
    if (urlArg) return urlArg.split('=')[1];

    const urlIdx = args.indexOf('--gateway');
    if (urlIdx !== -1 && args[urlIdx + 1]) return args[urlIdx + 1];

    return null;
}

function isRestore() {
    return process.argv.includes('--restore') || process.argv.includes('-r');
}

function backupFile(filePath) {
    if (!fs.existsSync(filePath)) return null;

    if (!fs.existsSync(BACKUP_DIR)) {
        fs.mkdirSync(BACKUP_DIR, { recursive: true });
    }

    const backupPath = path.join(BACKUP_DIR, path.basename(filePath));
    fs.copyFileSync(filePath, backupPath);
    console.log(`  ✓ Backed up: ${filePath} -> ${backupPath}`);
    return backupPath;
}

function patchSettings(gatewayUrl) {
    if (!fs.existsSync(SETTINGS_PATH)) {
        console.log('  ⚠ settings.json not found, creating new one');
        const settings = { [CONFIG_KEY]: gatewayUrl };
        fs.mkdirSync(path.dirname(SETTINGS_PATH), { recursive: true });
        fs.writeFileSync(SETTINGS_PATH, JSON.stringify(settings, null, 4));
        console.log(`  ✓ Created settings.json with gateway URL`);
        return true;
    }

    backupFile(SETTINGS_PATH);

    let settings;
    try {
        settings = JSON.parse(fs.readFileSync(SETTINGS_PATH, 'utf8'));
    } catch (e) {
        console.log('  ⚠ Failed to parse settings.json, creating new');
        settings = {};
    }

    const oldValue = settings[CONFIG_KEY];
    settings[CONFIG_KEY] = gatewayUrl;

    fs.writeFileSync(SETTINGS_PATH, JSON.stringify(settings, null, 4));

    if (oldValue) {
        console.log(`  ✓ Updated ${CONFIG_KEY}: "${oldValue}" -> "${gatewayUrl}"`);
    } else {
        console.log(`  ✓ Set ${CONFIG_KEY} = "${gatewayUrl}"`);
    }

    return true;
}

function patchExtension(gatewayUrl) {
    const extensionPath = '/opt/windsurf/resources/app/extensions/windsurf/dist/extension.js';

    if (!fs.existsSync(extensionPath)) {
        console.log('  ⚠ Windsurf extension not found at', extensionPath);
        console.log('  ℹ Skipping extension patch (settings.json patch is sufficient)');
        return false;
    }

    backupFile(extensionPath);

    let content = fs.readFileSync(extensionPath, 'utf8');

    const oldDefault = 'DEFAULT_API_SERVER_URL="https://server.codeium.com"';
    const newDefault = `DEFAULT_API_SERVER_URL="${gatewayUrl}"`;

    if (content.includes(oldDefault)) {
        content = content.replace(new RegExp(oldDefault.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'), newDefault);
        fs.writeFileSync(extensionPath, content);
        console.log(`  ✓ Patched extension.js: DEFAULT_API_SERVER_URL -> "${gatewayUrl}"`);
        return true;
    }

    console.log('  ⚠ Could not find DEFAULT_API_SERVER_URL in extension.js');
    return false;
}

function patchGlobalState(gatewayUrl) {
    const stateDbPath = path.join(WINDSURF_CONFIG_DIR, 'User', 'globalStorage', 'state.vscdb');

    if (!fs.existsSync(stateDbPath)) {
        console.log('  ℹ globalState database not found, skipping');
        return false;
    }

    try {
        const Database = require('better-sqlite3');
        backupFile(stateDbPath);

        const db = new Database(stateDbPath);
        const row = db.prepare('SELECT key, value FROM ItemTable WHERE key LIKE ?').get('%apiServerUrl%');

        if (row) {
            let value;
            try {
                value = JSON.parse(row.value);
            } catch (e) {
                value = row.value;
            }

            console.log(`  ✓ Found globalState entry: ${row.key}`);
            console.log(`    Old value: ${JSON.stringify(value)}`);

            const newValue = JSON.stringify(gatewayUrl);
            db.prepare('UPDATE ItemTable SET value = ? WHERE key = ?').run(newValue, row.key);
            console.log(`    Updated to: "${gatewayUrl}"`);
        } else {
            console.log('  ℹ No apiServerUrl entry in globalState');
        }

        db.close();
        return true;
    } catch (e) {
        console.log(`  ⚠ Failed to patch globalState: ${e.message}`);
        console.log('  ℹ This is optional, settings.json patch is sufficient');
        return false;
    }
}

function restore() {
    console.log('Restoring from backup...\n');

    if (!fs.existsSync(BACKUP_DIR)) {
        const dirs = fs.readdirSync(WINDSURF_CONFIG_DIR)
            .filter(d => d.startsWith('manual-auth-restore-'))
            .sort()
            .reverse();

        if (dirs.length === 0) {
            console.log('  ✗ No backup found');
            return;
        }

        const latestBackup = path.join(WINDSURF_CONFIG_DIR, dirs[0]);
        const files = fs.readdirSync(latestBackup);

        for (const file of files) {
            const src = path.join(latestBackup, file);

            if (file === 'settings.json') {
                fs.copyFileSync(src, SETTINGS_PATH);
                console.log(`  ✓ Restored settings.json`);
            } else if (file === 'extension.js') {
                const extPath = '/opt/windsurf/resources/app/extensions/windsurf/dist/extension.js';
                fs.copyFileSync(src, extPath);
                console.log(`  ✓ Restored extension.js`);
            } else if (file === 'state.vscdb') {
                const statePath = path.join(WINDSURF_CONFIG_DIR, 'User', 'globalStorage', 'state.vscdb');
                fs.copyFileSync(src, statePath);
                console.log(`  ✓ Restored globalState database`);
            }
        }

        console.log(`\n  ✓ Restored from backup: ${latestBackup}`);
        return;
    }

    console.log('  No backup directory found');
}

function main() {
    printBanner();

    if (isRestore()) {
        restore();
        return;
    }

    const gatewayUrl = getGatewayUrl();
    if (!gatewayUrl) {
        console.log('Usage:');
        console.log('  node patch.js --gateway=https://your-gateway.com');
        console.log('  WINDSURF_GATEWAY_URL=https://your-gateway.com node patch.js');
        console.log('  node patch.js --restore');
        console.log();
        console.log('Environment variables:');
        console.log('  WINDSURF_GATEWAY_URL  - Gateway server URL');
        process.exit(1);
    }

    console.log(`Gateway URL: ${gatewayUrl}`);
    console.log(`Windsurf config: ${WINDSURF_CONFIG_DIR}\n`);

    console.log('[1/3] Patching settings.json...');
    patchSettings(gatewayUrl);

    console.log('\n[2/3] Patching extension.js...');
    patchExtension(gatewayUrl);

    console.log('\n[3/3] Patching globalState...');
    patchGlobalState(gatewayUrl);

    console.log(`\n✓ Patch complete! Backups saved to: ${BACKUP_DIR}`);
    console.log('  Restart Windsurf for changes to take effect.');
    console.log('  To restore: node patch.js --restore\n');
}

main();
