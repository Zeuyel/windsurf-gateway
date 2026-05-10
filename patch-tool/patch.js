#!/usr/bin/env node

const crypto = require('crypto')
const fs = require('fs')
const path = require('path')
const os = require('os')

const DEFAULT_API_URL = 'https://server.codeium.com'
const CONFIG_KEY = 'codeium.apiServerUrl'
const REGISTER_CONFIG_KEY = 'codeium.registerApiServerUrl'
const DEFAULT_REGISTER_URL = 'https://register.windsurf.com'
const LEGACY_GATEWAY_PLACEHOLDER_API_KEY = 'sk-ws-01-gateway-placeholder'
const GATEWAY_PLACEHOLDER_PREFIX = 'sk-ws-01-client-'
const ACP_CONFIG_KEY = 'windsurf.acp.enabledAgents'
const PATCH_STATE_PATH = path.join('User', 'globalStorage', 'windsurf-gateway-patch.json')
const AUTH_SESSION_FALLBACK_MARKER = 'account:{label:"Gateway",id:"windsurf-gateway"},scopes:[]}'
const AUTH_SESSION_FALLBACK_BLOCK_REGEX = /\?\?\{id:"windsurf-gateway",accessToken:"[^"]+",account:\{label:"Gateway",id:"windsurf-gateway"\},scopes:\[\]\}/
const AUTH_SESSION_FALLBACK_REGEX = /await i\.authentication\.getSession\(n\.WindsurfExtensionMetadata\.getInstance\(\)\.authProviderId,\[[\s\S]*?\],e\)/
const USER_STATUS_FALLBACK_SENTINEL = `allowedCommandModelConfigsProtoBinaryBase64:[],userStatusProtoBinaryBase64:""}`
const USER_STATUS_FALLBACK_REGEX = /([A-Za-z_$][\w$]*)\.StatusBar\.getInstance\(\)\.setAuthStatus\(!1\),([A-Za-z_$][\w$]*)\.windsurfAuth\.setAuthStatus\(null\),\(await\(0,([A-Za-z_$][\w$]*)\.getAuthSession\)\(\)\)\?\.accessToken===([A-Za-z_$][\w$]*)\|\|([A-Za-z_$][\w$]*)\.clearAuthentication\(\),!1/

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
const patchStateFile = path.join(configDir, PATCH_STATE_PATH)

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

function loadPatchState() {
  if (!fs.existsSync(patchStateFile)) return {}
  try {
    return JSON.parse(fs.readFileSync(patchStateFile, 'utf8') || '{}')
  } catch (e) {
    console.log(`patch state unreadable, regenerating: ${patchStateFile}`)
    return {}
  }
}

function savePatchState(state) {
  writeJson(patchStateFile, state)
}

function generatePlaceholderApiKey() {
  return GATEWAY_PLACEHOLDER_PREFIX + crypto.randomBytes(18).toString('hex')
}

function ensurePatchIdentity() {
  const state = loadPatchState()
  if (typeof state.placeholderApiKey !== 'string' || !state.placeholderApiKey) {
    state.placeholderApiKey = generatePlaceholderApiKey()
    state.createdAt = new Date().toISOString()
    savePatchState(state)
  }
  return state
}

function placeholderSummary(value) {
  if (!value) return '(missing)'
  return `${value.slice(0, 18)}...${value.slice(-6)}`
}

function patchSettings(gateway, registerGateway, identity) {
  backup(settingsPath)
  const settings = readJson(settingsPath)
  const oldApi = settings[CONFIG_KEY]
  const oldRegister = settings[REGISTER_CONFIG_KEY]
  const enabledAgents = typeof settings[ACP_CONFIG_KEY] === 'object' && settings[ACP_CONFIG_KEY] !== null
    ? settings[ACP_CONFIG_KEY]
    : {}
  settings[CONFIG_KEY] = gateway
  if (registerGateway) settings[REGISTER_CONFIG_KEY] = registerGateway
  settings[ACP_CONFIG_KEY] = { ...enabledAgents, 'devin-cloud': false }
  writeJson(settingsPath, settings)
  console.log(`settings ${CONFIG_KEY}: ${oldApi || DEFAULT_API_URL} -> ${gateway}`)
  if (registerGateway) console.log(`settings ${REGISTER_CONFIG_KEY}: ${oldRegister || DEFAULT_REGISTER_URL} -> ${registerGateway}`)
  console.log(`settings ${ACP_CONFIG_KEY}.devin-cloud -> false`)
  if (identity?.placeholderApiKey) {
    console.log(`gateway placeholder key: ${placeholderSummary(identity.placeholderApiKey)}`)
  }
}

function buildAuthSessionFallbackBlock(placeholderApiKey) {
  return `??{id:"windsurf-gateway",accessToken:"${placeholderApiKey}",account:{label:"Gateway",id:"windsurf-gateway"},scopes:[]}`
}

function patchExtension(gateway, registerGateway, identity) {
  if (!fs.existsSync(extensionPath)) {
    console.log(`extension not found: ${extensionPath}`)
    return false
  }
  backup(extensionPath)
  let content = fs.readFileSync(extensionPath, 'utf8')
  const before = content
  const placeholderApiKey = identity?.placeholderApiKey || LEGACY_GATEWAY_PLACEHOLDER_API_KEY
  content = content.replaceAll(`DEFAULT_API_SERVER_URL=\"${DEFAULT_API_URL}\"`, `DEFAULT_API_SERVER_URL=\"${gateway}\"`)
  content = content.replaceAll(`DEFAULT_API_SERVER_URL="${DEFAULT_API_URL}"`, `DEFAULT_API_SERVER_URL="${gateway}"`)
  if (registerGateway) {
    content = content.replaceAll(`DEFAULT_REGISTER_API_SERVER_URL=\"${DEFAULT_REGISTER_URL}\"`, `DEFAULT_REGISTER_API_SERVER_URL=\"${registerGateway}\"`)
    content = content.replaceAll(`DEFAULT_REGISTER_API_SERVER_URL="${DEFAULT_REGISTER_URL}"`, `DEFAULT_REGISTER_API_SERVER_URL="${registerGateway}"`)
  }
  if (AUTH_SESSION_FALLBACK_BLOCK_REGEX.test(content)) {
    content = content.replace(AUTH_SESSION_FALLBACK_BLOCK_REGEX, buildAuthSessionFallbackBlock(placeholderApiKey))
  } else {
    content = content.replace(AUTH_SESSION_FALLBACK_REGEX, (match) => {
      return `${match}${buildAuthSessionFallbackBlock(placeholderApiKey)}`
    })
  }
  if (!content.includes(USER_STATUS_FALLBACK_SENTINEL)) {
    content = content.replace(USER_STATUS_FALLBACK_REGEX, (_match, statusBarRef, authRef, _sessionRef, apiKeyRef) => {
      return `${statusBarRef}.StatusBar.getInstance().setAuthStatus(!0),${authRef}.windsurfAuth.setAuthStatus({apiKey:${apiKeyRef},allowedCommandModelConfigsProtoBinaryBase64:[],userStatusProtoBinaryBase64:""}),!0`
    })
  }
  if (content.includes(AUTH_SESSION_FALLBACK_MARKER) && !before.includes(AUTH_SESSION_FALLBACK_MARKER)) {
    console.log('extension auth bootstrap patched with gateway fallback session')
  }
  if (content.includes(USER_STATUS_FALLBACK_SENTINEL) && !before.includes(USER_STATUS_FALLBACK_SENTINEL)) {
    console.log('extension user-status fallback patched for gateway mode')
  }
  if (content === before) {
    console.log('extension already patched or patch markers not found')
    return false
  }
  fs.writeFileSync(extensionPath, content)
  console.log(`extension patched: ${extensionPath}`)
  console.log(`extension placeholder key: ${placeholderSummary(placeholderApiKey)}`)
  return true
}

function patchGlobalState(gateway, identity) {
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
  
  // Patch API server URLs
  const rows = db.prepare('SELECT key,value FROM ItemTable WHERE key LIKE ? OR key LIKE ?').all('%apiServerUrl%', '%BASE_API_SERVER_URL%')
  for (const row of rows) {
    db.prepare('UPDATE ItemTable SET value = ? WHERE key = ?').run(JSON.stringify(gateway), row.key)
    console.log(`globalState ${row.key} -> ${gateway}`)
  }
  
  // Mock windsurfAuthStatus to skip signup
  const mockAuthStatus = {
    apiKey: identity?.placeholderApiKey || LEGACY_GATEWAY_PLACEHOLDER_API_KEY,
    allowedCommandModelConfigsProtoBinaryBase64: [],
    userStatusProtoBinaryBase64: ''
  }
  db.prepare(`
    INSERT INTO ItemTable (key, value) VALUES (?, ?)
    ON CONFLICT(key) DO UPDATE SET value = excluded.value
  `).run('windsurfAuthStatus', JSON.stringify(mockAuthStatus))
  console.log('globalState windsurfAuthStatus -> mocked')
  console.log(`globalState placeholder key: ${placeholderSummary(mockAuthStatus.apiKey)}`)
  
  // Mock onboarding completion
  db.prepare(`
    INSERT INTO ItemTable (key, value) VALUES (?, ?)
    ON CONFLICT(key) DO UPDATE SET value = excluded.value
  `).run('windsurfOnboarding', JSON.stringify(true))
  console.log('globalState windsurfOnboarding -> true')
  
  // Mock product education completed
  const mockProductEducation = {
    onboardingState: 2,
    onboardingItems: [
      {
        id: 'windsurf.prioritized.chat.open',
        title: 'Code with Cascade',
        completed: true,
        command: 'windsurf.prioritized.chat.open'
      }
    ]
  }
  db.prepare(`
    INSERT INTO ItemTable (key, value) VALUES (?, ?)
    ON CONFLICT(key) DO UPDATE SET value = excluded.value
  `).run('windsurfProductEducation', JSON.stringify(mockProductEducation))
  console.log('globalState windsurfProductEducation -> mocked')
  
  db.close()
  return rows.length > 0
}

function detect() {
  const state = loadPatchState()
  console.log(`configDir: ${configDir}`)
  console.log(`installDir: ${installDir}`)
  console.log(`settings: ${fs.existsSync(settingsPath) ? settingsPath : 'not found'}`)
  console.log(`globalState: ${fs.existsSync(globalStatePath) ? globalStatePath : 'not found'}`)
  console.log(`extension: ${fs.existsSync(extensionPath) ? extensionPath : 'not found'}`)
  console.log(`patchState: ${fs.existsSync(patchStateFile) ? patchStateFile : 'not found'}`)
  console.log(`placeholder key: ${placeholderSummary(state.placeholderApiKey)}`)
  console.log(`placeholder mode: ${state.placeholderApiKey === LEGACY_GATEWAY_PLACEHOLDER_API_KEY ? 'legacy-shared' : 'per-client'}`)
  if (fs.existsSync(settingsPath)) {
    const settings = readJson(settingsPath)
    console.log(`${CONFIG_KEY}: ${settings[CONFIG_KEY] || '(default)'}`)
    console.log(`${REGISTER_CONFIG_KEY}: ${settings[REGISTER_CONFIG_KEY] || '(default)'}`)
    console.log(`${ACP_CONFIG_KEY}.devin-cloud: ${settings[ACP_CONFIG_KEY]?.['devin-cloud']}`)
  }
  if (fs.existsSync(extensionPath)) {
    const content = fs.readFileSync(extensionPath, 'utf8')
    console.log(`contains ${DEFAULT_API_URL}: ${content.includes(DEFAULT_API_URL)}`)
    console.log(`contains ${DEFAULT_REGISTER_URL}: ${content.includes(DEFAULT_REGISTER_URL)}`)
    console.log(`contains auth-session fallback: ${AUTH_SESSION_FALLBACK_BLOCK_REGEX.test(content)}`)
    console.log(`contains user-status fallback: ${content.includes(USER_STATUS_FALLBACK_SENTINEL)}`)
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
  
  // Clean up mocked data from globalState
  let Database
  try {
    Database = require('better-sqlite3')
  } catch (e) {
    console.log('better-sqlite3 not installed; skip cleanup')
    return
  }
  
  if (fs.existsSync(globalStatePath)) {
    const db = new Database(globalStatePath)
    const mockKeys = ['windsurfAuthStatus', 'windsurfOnboarding', 'windsurfProductEducation']
    for (const key of mockKeys) {
      const row = db.prepare('SELECT key FROM ItemTable WHERE key = ?').get(key)
      if (row) {
        db.prepare('DELETE FROM ItemTable WHERE key = ?').run(key)
        console.log(`cleaned up ${key}`)
      }
    }
    db.close()
  }
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
    const identity = ensurePatchIdentity()
    patchSettings(gateway, registerGateway, identity)
    patchGlobalState(gateway, identity)
  }
  if (mode === 'extension' || mode === 'all') {
    const identity = ensurePatchIdentity()
    patchExtension(gateway, registerGateway, identity)
  }
  console.log('\npatch done. Restart Windsurf.')
}

main()
