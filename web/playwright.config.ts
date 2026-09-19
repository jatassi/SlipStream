import { defineConfig, type PlaywrightTestConfig, type Project } from '@playwright/test'
import { mkdirSync } from 'node:fs'
import path from 'node:path'

import { authStatePath, backendOrigin, e2eDataDir, frontendOrigin, repoRoot, webDir } from './e2e/helpers/paths'

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
    testMatch: /shell\.spec\.ts/,
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
  return env
}

const config: PlaywrightTestConfig = {
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: inCI,
  retries: inCI ? 1 : 0,
  workers: inCI ? 2 : undefined,
  reporter: inCI ? [['github'], ['html', { open: 'never' }]] : [['list']],
  timeout: 30_000,
  expect: { timeout: 15_000 },
  snapshotPathTemplate: '{testDir}/snapshots/{projectName}/{testFileName}-{arg}{ext}',
  use: {
    baseURL: frontendOrigin,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  webServer: [
    {
      command: 'go run ./cmd/slipstream --config configs/config.example.yaml --dev-mode --no-tray',
      cwd: repoRoot,
      url: `${backendOrigin}/health`,
      reuseExistingServer: !inCI,
      timeout: 180_000,
      stdout: 'pipe',
      stderr: 'pipe',
      env: backendEnv(),
    },
    {
      command: 'bun run dev',
      cwd: webDir,
      url: frontendOrigin,
      reuseExistingServer: !inCI,
      timeout: 60_000,
    },
  ],
  projects: [setupProject(), ...shellProjects(), reducedMotionProject()],
}

export default defineConfig(config)
