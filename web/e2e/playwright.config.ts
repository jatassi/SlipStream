import { join } from "node:path";
import { arch, env, platform } from "node:process";

import { type Project, defineConfig } from "@playwright/test";

import presets from "../harness/presets.json" with { type: "json" };

type Engine = "chromium" | "firefox" | "webkit";

// Playwright refuses isMobile on Firefox, so there is no firefox-phone.
const projects = {
  "chromium-desk": ["chromium", "desk"],
  "chromium-phone": ["chromium", "phone"],
  "firefox-desk": ["firefox", "desk"],
  "webkit-desk": ["webkit", "desk"],
  "webkit-phone": ["webkit", "phone"],
} as const satisfies Record<string, readonly [Engine, keyof typeof presets]>;

type ProjectName = keyof typeof projects;

export const targets = {
  "linux-amd64": ["chromium-desk", "chromium-phone", "firefox-desk", "webkit-phone"],
  "linux-arm64": ["chromium-desk"],
  "darwin-arm64": ["chromium-desk", "webkit-desk", "webkit-phone"],
  "windows-amd64": ["chromium-desk", "firefox-desk"],
} as const satisfies Record<string, readonly ProjectName[]>;

const goos: Partial<Record<NodeJS.Platform, string>> = {
  linux: "linux",
  darwin: "darwin",
  win32: "windows",
};
const goarch: Partial<Record<NodeJS.Architecture, string>> = { x64: "amd64", arm64: "arm64" };

const isTarget = (target: string): target is keyof typeof targets => Object.hasOwn(targets, target);

export function targetProjects(): readonly ProjectName[] {
  const target = `${goos[platform]}-${goarch[arch]}`;
  if (!isTarget(target)) {
    throw new Error(`the browser tier has no projects for ${platform}-${arch}`);
  }
  return targets[target];
}

export const engineOf = (name: ProjectName): Engine => projects[name][0];

function project(name: ProjectName): Project {
  const [engine, preset] = projects[name];
  const { width, height, ...device } = presets[preset];
  return {
    name,
    use: { browserName: engine, contextOptions: { viewport: { width, height }, ...device } },
  };
}

const runs = env["TEST_FAKEWORLD_RUNS"];

export default defineConfig({
  testDir: ".",
  forbidOnly: true,
  retries: 0,
  fullyParallel: true,
  use: { trace: "retain-on-failure" },
  // Playwright empties its outputDir at start, and the Go tier leaves its
  // failed run dirs directly under TEST_FAKEWORLD_RUNS.
  ...(runs === undefined ? {} : { outputDir: join(runs, "playwright") }),
  projects: targetProjects().map((name) => project(name)),
});
