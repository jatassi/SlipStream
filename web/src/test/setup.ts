import "../styles.css";

import { faker } from "@faker-js/faker";
import { afterAll, beforeAll, beforeEach, onTestFinished } from "vitest";

import { writePref } from "../shell/prefs";
import { unmountApps } from "./render-app";
import { worker } from "./worker";

declare module "vitest/browser" {
  interface BrowserCommands {
    emulateReducedMotion: (reduce: boolean) => Promise<void>;
  }
}

beforeAll(async () => {
  await worker.start({
    quiet: true,
    onUnhandledFrame: "error",
  });
});

// onTestFinished hooks run last-registered first: a test's own hooks release
// held requests, then its apps settle and unmount, then its handlers go.
beforeEach(() => {
  faker.seed(6);
  writePref("startupPlayed", true);
  writePref("sidebarCollapsed", false);
  onTestFinished(() => {
    worker.resetHandlers();
  });
  onTestFinished(unmountApps);
});

afterAll(async () => {
  await worker.stop();
});
