#!/usr/bin/env node

const fs = require('fs')
const path = require('path')
const os = require('os')

const DEFAULT_API_URL = 'https://server.codeium.com'
const CONFIG_KEY = 'codeium.apiServerUrl'
const REGISTER_CONFIG_KEY = 'codeium.registerApiServerUrl'
const DEFAULT_REGISTER_URL = 'https://register.windsurf.com'

function argValue(name) {
  const args = process.argv.slice(2)
  const eq = args.find(a => a.startsWith(`${name}=`))
  if (eq) return eq.slice(name.length + 1)
  const idx = args.indexOf(name)
  return idx >= 0 ? args[idx + 1] : null
}

function hasArg(name) {
  return process.argv.includes(name)
}

function homeConfigDir() {
  if (process.env.WINDSURF_CONFIG_DIR) return process.env.WINDSURF_CONFIG_DIR
  if (process.platform === 'darwin') return path.join(os.homedir(), 'Library', 'Application Support', 'Windsurf')
  if (process.platform === 'win32') return path.join(process.env.APPDATA || path.join(os.homedir(), 'AppData', 'Roaming'), 'Windsurf')
  return path.join(os.homedir(), '.config', 'Windsurf')
}

function defaultInstallDir() {
  if (process.env.WINDSURF_INSTALL_DIR) return process.env.WINDSURF_INSTALL_DIR
  if (process.platform === 'darwin') return '/Applications/Windsurf.app/Contents/Resources/app'
  if (process.platform === 'win32') return path.join(process.env.LOCALAPPDATA || '', 'Programs', 'Windsurf', 'resources', 'app')
  return '/opt/windsurf/resources/app'
}

const configDir = homeConfigDir()
const installDir = defaultInstallDir()
const settingsPath = path.join(configDir, 'User', 'settings.json')
const globalStatePath = path.join(configDir, 'User', 'globalStorage', 'state.vscdb')
const extensionPath = path.join(installDir, 'extensions', 'windsurf', 'dist', 'extension.js')
const backupRoot = path.join(configDir, 'windsurf-gateway-backups')

function banner() {
  console.log('\nWindsurf Gateway Patch Tool\n')
}

function backup(filePath) {
  if (!fs.existsSync(filePath)) return null
  const stamp = new Date().toISOString().replace(/[:.]/g, '-')
  const dir = path.join(backupRoot, stamp)
  fs.mkdirSync(dir, { recursive: true })
  const out = path.join(dir, path.basename(filePath))
  fs.copyFileSync(filePath, out)
  console.log(`backup ${filePath} -> ${out}`)
  return out
}

function readJson(filePath) {
  if (!fs.existsSync(filePath)) return {}
  return JSON.parse(fs.readFileSync(filePath, 'utf8') || '{}')
}

function writeJson(filePath, data) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true })
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + '\n')
}

function patchSettings(gateway, registerGateway) {
  backup(settingsPath)
  const settings = readJson(settingsPath)
  const oldApi = settings[CONFIG_KEY]
  const oldRegister = settings[REGISTER_CONFIG_KEY]
  settings[CONFIG_KEY] = gateway
  if (registerGateway) settings[REGISTER_CONFIG_KEY] = registerGateway
  writeJson(settingsPath, settings)
  console.log(`settings ${CONFIG_KEY}: ${oldApi || DEFAULT_API_URL} -> ${gateway}`)
  if (registerGateway) console.log(`settings ${REGISTER_CONFIG_KEY}: ${oldRegister || DEFAULT_REGISTER_URL} -> ${registerGateway}`)
}

function patchExtension(gateway, registerGateway) {
  if (!fs.existsSync(extensionPath)) {
    console.log(`extension not found: ${extensionPath}`)
    return false
  }
  backup(extensionPath)
  let content = fs.readFileSync(extensionPath, 'utf8')
  const before = content
  content = content.replaceAll(`DEFAULT_API_SERVER_URL=\"${DEFAULT_API_URL}\"`, `DEFAULT_API_SERVER_URL=\"${gateway}\"`)
  content = content.replaceAll(`DEFAULT_API_SERVER_URL="${DEFAULT_API_URL}"`, `DEFAULT_API_SERVER_URL="${gateway}"`)
  if (registerGateway) {
    content = content.replaceAll(`DEFAULT_REGISTER_API_SERVER_URL=\"${DEFAULT_REGISTER_URL}\"`, `DEFAULT_REGISTER_API_SERVER_URL=\"${registerGateway}\"`)
    content = content.replaceAll(`DEFAULT_REGISTER_API_SERVER_URL="${DEFAULT_REGISTER_URL}"`, `DEFAULT_REGISTER_API_SERVER_URL="${registerGateway}"`)
  }
  if (content === before) {
    console.log('extension constants not changed; default endpoint pattern not found')
    return false
  }
  fs.writeFileSync(extensionPath, content)
  console.log(`extension patched: ${extensionPath}`)
  return true
}

function patchGlobalState(gateway) {
  if (!fs.existsSync(globalStatePath)) {
    console.log(`globalState not found: ${globalStatePath}`)
    return false
  }
  let Database
  try {
    Database = require('better-sqlite3')
  } catch (e) {
    console.log('better-sqlite3 not installed; skip globalState')
    return false
  }
  backup(globalStatePath)
  const db = new Database(globalStatePath)
  const rows = db.prepare('SELECT key,value FROM ItemTable WHERE key LIKE ? OR key LIKE ?').all('%apiServerUrl%', '%BASE_API_SERVER_URL%')
  for (const row of rows) {
    db.prepare('UPDATE ItemTable SET value = ? WHERE key = ?').run(JSON.stringify(gateway), row.key)
    console.log(`globalState ${row.key} -> ${gateway}`)
  }
  db.close()
  return rows.length > 0
}

function detect() {
  console.log(`configDir: ${configDir}`)
  console.log(`installDir: ${installDir}`)
  console.log(`settings: ${fs.existsSync(settingsPath) ? settingsPath : 'not found'}`)
  console.log(`globalState: ${fs.existsSync(globalStatePath) ? globalStatePath : 'not found'}`)
  console.log(`extension: ${fs.existsSync(extensionPath) ? extensionPath : 'not found'}`)
  if (fs.existsSync(settingsPath)) {
    const settings = readJson(settingsPath)
    console.log(`${CONFIG_KEY}: ${settings[CONFIG_KEY] || '(default)'}`)
    console.log(`${REGISTER_CONFIG_KEY}: ${settings[REGISTER_CONFIG_KEY] || '(default)'}`)
  }
  if (fs.existsSync(extensionPath)) {
    const content = fs.readFileSync(extensionPath, 'utf8')
    console.log(`contains ${DEFAULT_API_URL}: ${content.includes(DEFAULT_API_URL)}`)
    console.log(`contains ${DEFAULT_REGISTER_URL}: ${content.includes(DEFAULT_REGISTER_URL)}`)
  }
}

function restore() {
  if (!fs.existsSync(backupRoot)) {
    console.log('no backup root found')
    return
  }
  const dirs = fs.readdirSync(backupRoot).sort().reverse()
  if (dirs.length === 0) {
    console.log('no backups found')
    return
  }
  const dir = path.join(backupRoot, dirs[0])
  const files = fs.readdirSync(dir)
  for (const file of files) {
    const src = path.join(dir, file)
    if (file === 'settings.json') fs.copyFileSync(src, settingsPath)
    if (file === 'extension.js') fs.copyFileSync(src, extensionPath)
    if (file === 'state.vscdb') fs.copyFileSync(src, globalStatePath)
    console.log(`restored ${file}`)
  }
  console.log(`restored from ${dir}`)
}

function usage() {
  console.log('Usage:')
  console.log('  node patch.js detect')
  console.log('  node patch.js --gateway=https://gateway.example.com [--mode=config|extension|all]')
  console.log('  node patch.js --restore')
  console.log('Env:')
  console.log('  WINDSURF_GATEWAY_URL=https://gateway.example.com')
  console.log('  WINDSURF_CONFIG_DIR=/custom/config')
  console.log('  WINDSURF_INSTALL_DIR=/custom/resources/app')
}

function main() {
  banner()
  if (process.argv.includes('detect') || hasArg('--detect')) return detect()
  if (hasArg('--restore') || hasArg('-r')) return restore()
  const gateway = argValue('--gateway') || process.env.WINDSURF_GATEWAY_URL
  const registerGateway = argValue('--register-gateway') || process.env.WINDSURF_REGISTER_GATEWAY_URL
  const mode = argValue('--mode') || 'all'
  if (!gateway) return usage()
  if (!/^https?:\/\//.test(gateway)) throw new Error('gateway must start with http:// or https://')
  if (mode === 'config' || mode === 'all') {
    patchSettings(gateway, registerGateway)
    patchGlobalState(gateway)
  }
  if (mode === 'extension' || mode === 'all') patchExtension(gateway, registerGateway)
  console.log('\npatch done. Restart Windsurf.')
}

main()
