import "../styles.css";

import { faker } from "@faker-js/faker";
import { afterAll, beforeAll, beforeEach, onTestFinished } from "vitest";
import { commands } from "vitest/browser";

import { writePref } from "../shell/prefs";
import { unmountApps } from "./render-app";
import { restoreViewport } from "./viewport";
import { worker } from "./worker";

declare module "vitest/browser" {
  interface BrowserCommands {
    holdServiceWorker: () => Promise<void>;
    emulateReducedMotion: (reduce: boolean) => Promise<void>;
    drag: (from: string, to: string) => Promise<void>;
    swipe: (
      from: { selector: string; at: readonly [number, number] },
      by: readonly [number, number],
      finger: boolean,
    ) => Promise<void>;
    wheel: (
      over: { selector: string; at: readonly [number, number] },
      deltaY: number,
    ) => Promise<void>;
    pinch: (
      centre: { selector: string; at: readonly [number, number] },
      from: number,
      to: number,
    ) => Promise<void>;
  }
}

beforeAll(async () => {
  await worker.start({
    quiet: true,
    onUnhandledFrame: "error",
  });
  await commands.holdServiceWorker();
});

// onTestFinished hooks run last-registered first: a test's own hooks release
// held requests, then its apps settle and unmount, then its handlers go.
beforeEach(() => {
  onTestFinished(restoreViewport);
  faker.seed(6);
  writePref("startupPlayed", true);
  writePref("sidebarCollapsed", false);
  writePref("startupOff", false);
  writePref("lastEmail", "");
  writePref("showHiddenFolders", false);
  writePref("libraryViews", {});
  onTestFinished(() => {
    worker.resetHandlers();
  });
  onTestFinished(unmountApps);
});

afterAll(async () => {
  await worker.stop();
});
