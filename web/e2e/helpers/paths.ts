import path from 'node:path'
import { fileURLToPath } from 'node:url'

const helpersDir = path.dirname(fileURLToPath(import.meta.url))

export const e2eDir = path.resolve(helpersDir, '..')
export const webDir = path.resolve(e2eDir, '..')
export const repoRoot = path.resolve(webDir, '..')
export const authStatePath = path.join(e2eDir, '.auth', 'admin.json')
export const e2eDataDir = path.join(e2eDir, '.data')

export const frontendOrigin = 'http://127.0.0.1:3000'
export const backendOrigin = 'http://127.0.0.1:8080'
export const apiBase = `${backendOrigin}/api/v1`

export const adminPin = '1234'
export const adminUsername = 'Administrator'
export const authStorageKey = 'slipstream-portal-auth'
