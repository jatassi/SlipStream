import { execFileSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const helpersDir = path.dirname(fileURLToPath(import.meta.url))

export const e2eDir = path.resolve(helpersDir, '..')
export const webDir = path.resolve(e2eDir, '..')
export const repoRoot = path.resolve(webDir, '..')
export const authStatePath = path.join(e2eDir, '.auth', 'admin.json')
export const e2eDataDir = path.join(e2eDir, '.data')

// Locally every checkout (including each git worktree) gets its own named
// portless route, so parallel suites never contend for a port. CI has no
// portless proxy and runs one suite on fixed ports.
export const usePortless = !process.env.CI

export const frontendService = 'slipstream-e2e'
export const backendService = 'slipstream-e2e-api'

function portlessUrl(service: string): string {
  return execFileSync('portless', ['get', service], { cwd: webDir, encoding: 'utf8' }).trim()
}

export const frontendOrigin = usePortless ? portlessUrl(frontendService) : 'http://127.0.0.1:3000'
export const backendOrigin = usePortless ? portlessUrl(backendService) : 'http://127.0.0.1:8080'
export const apiBase = `${backendOrigin}/api/v1`

export const adminPin = '1234'
export const adminUsername = 'Administrator'
export const authStorageKey = 'slipstream-portal-auth'
