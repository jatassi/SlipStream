import { defineConfig, type PlaywrightTestConfig, type Project } from '@playwright/test'
import { mkdirSync } from 'node:fs'
import path from 'node:path'

import {
  authStatePath,
  backendOrigin,
  backendService,
  e2eDataDir,
  frontendOrigin,
  frontendService,
  repoRoot,
  usePortless,
  webDir,
} from './e2e/helpers/paths'

const inCI = Boolean(process.env.CI)

mkdirSync(e2eDataDir, { recursive: true })

const phoneUse = {
  storageState: authStatePath,
  viewport: { width: 390, height: 844 },
  hasTouch: true,
  isMobile: true,
  deviceScaleFactor: 2,
}

const wideUse = {
  storageState: authStatePath,
  viewport: { width: 1440, height: 900 },
  hasTouch: false,
  isMobile: false,
  deviceScaleFactor: 1,
}

function setupProject(): Project {
  return { name: 'setup', testMatch: /auth\.setup\.ts/ }
}

function shellProjects(): Project[] {
  return [
    { name: 'phone', dependencies: ['setup'], testIgnore: /auth\.setup\.ts/, use: phoneUse },
    { name: 'wide', dependencies: ['setup'], testIgnore: /auth\.setup\.ts/, use: wideUse },
  ]
}

function reducedMotionProject(): Project {
  return {
    name: 'reduced-motion',
    dependencies: ['setup'],
    testMatch: /(shell|push)\.spec\.ts/,
    use: { ...phoneUse, reducedMotion: 'reduce' },
  }
}

function backendEnv(): Record<string, string> {
  const env: Record<string, string> = {}
  for (const [key, value] of Object.entries(process.env)) {
    if (value !== undefined) {
      env[key] = value
    }
  }
  env.SLIPSTREAM_DATABASE_PATH = path.join(e2eDataDir, 'slipstream.db')
  env.SLIPSTREAM_INDEXER_CARDIGANN_AUTO_UPDATE = 'false'
  env.SLIPSTREAM_LOGGING_LEVEL = 'warn'
  env.SLIPSTREAM_LOGGING_PATH = path.join(e2eDataDir, 'logs')
  env.SLIPSTREAM_SERVER_HOST = '127.0.0.1'
  env.SLIPSTREAM_DEV_BUILD = '1'
  return env
}

const GRACEFUL_SHUTDOWN = { signal: 'SIGTERM', timeout: 15_000 } as const

const BACKEND_BINARY = 'web/e2e/.data/slipstream'
const BACKEND_ARGS = '--config configs/config.example.yaml --dev-mode --no-tray'

// Playwright stops a server by signalling its process group and waiting for its
// stdio to close. portless starts the real server in a session of its own and
// forwards SIGTERM to it, so the servers must be stopped with SIGTERM (the
// default SIGKILL never reaches them and orphans them holding the pipe), and
// nothing that swallows SIGTERM may sit in between: the backend is built once
// and exec'd directly (not `go run`) and Vite runs through its own binary (not
// `bun run`). Under portless each server is started through the
// proxy under a per-worktree name and receives its real port through PORT; the
// backend is told to listen on it and Vite reads PORT itself.
function backendCommand(): string {
  const build = `go build -o ${BACKEND_BINARY} ./cmd/slipstream`
  if (!usePortless) {
    return `${build} && ${BACKEND_BINARY} ${BACKEND_ARGS}`
  }
  return `${build} && portless run --name ${backendService} sh -c 'SLIPSTREAM_SERVER_PORT="$PORT" exec ${BACKEND_BINARY} ${BACKEND_ARGS}'`
}

function frontendCommand(): string {
  const vite = 'node_modules/.bin/vite'
  return usePortless ? `portless run --name ${frontendService} ${vite}` : vite
}

const config: PlaywrightTestConfig = {
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: inCI,
  retries: inCI ? 1 : 0,
  workers: inCI ? 1 : undefined,
  reporter: inCI ? [['github'], ['html', { open: 'never' }]] : [['list']],
  timeout: 30_000,
  expect: { timeout: 15_000 },
  snapshotPathTemplate: '{testDir}/snapshots/{projectName}/{testFileName}-{arg}{ext}',
  use: {
    baseURL: frontendOrigin,
    ignoreHTTPSErrors: usePortless,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  webServer: [
    {
      command: backendCommand(),
      cwd: repoRoot,
      url: `${backendOrigin}/health`,
      ignoreHTTPSErrors: usePortless,
      reuseExistingServer: false,
      gracefulShutdown: GRACEFUL_SHUTDOWN,
      timeout: 180_000,
      stdout: 'pipe',
      stderr: 'pipe',
      env: backendEnv(),
    },
    {
      command: frontendCommand(),
      cwd: webDir,
      url: frontendOrigin,
      ignoreHTTPSErrors: usePortless,
      reuseExistingServer: false,
      gracefulShutdown: GRACEFUL_SHUTDOWN,
      timeout: 60_000,
      env: { ...process.env, SLIPSTREAM_API_ORIGIN: backendOrigin },
    },
  ],
  projects: [setupProject(), ...shellProjects(), reducedMotionProject()],
}

export default defineConfig(config)
