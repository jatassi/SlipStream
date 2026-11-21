import { expect, onTestFinished, test, vi } from "vitest";
import { page } from "vitest/browser";

import { frames } from "../test/frames";
import { renderKit } from "../test/render-kit";
import { type Tick, useNow } from "./clock";

function useFakeClock(now: string) {
  vi.useFakeTimers({ toFake: ["Date", "setTimeout", "clearTimeout"], now: new Date(now) });
  onTestFinished(() => {
    vi.useRealTimers();
  });
}

function Reader({ name, tick }: { name: string; tick?: Tick }) {
  return <p aria-label={name}>{useNow(tick).toISOString()}</p>;
}

// Polling would advance the fake clock, so each check waits real frames for
// React to subscribe and render, then reads once.
async function read(name: string) {
  await frames(3);
  return page.getByLabelText(name).element().textContent;
}

function setVisibility(state: DocumentVisibilityState) {
  Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
  document.dispatchEvent(new Event("visibilitychange"));
}

test("every minute reader turns on the wall minute, to one now", async () => {
  useFakeClock("2026-11-21T14:02:30.000Z");
  renderKit(
    <>
      <Reader name="Age" />
      <Reader name="Clock" />
    </>,
  );
  expect(await read("Age")).toBe("2026-11-21T14:02:30.000Z");

  await vi.advanceTimersByTimeAsync(29_999);
  expect(await read("Clock")).toBe("2026-11-21T14:02:30.000Z");
  await vi.advanceTimersByTimeAsync(1);
  expect([await read("Age"), await read("Clock")]).toEqual([
    "2026-11-21T14:03:00.000Z",
    "2026-11-21T14:03:00.000Z",
  ]);
  expect(vi.getTimerCount()).toBe(1);
});

test("a second reader ticks each second while it is shown, and minute readers still turn on the minute", async () => {
  useFakeClock("2026-11-21T14:02:58.000Z");
  renderKit(
    <>
      <Reader name="Code" tick="second" />
      <Reader name="Age" />
    </>,
  );
  expect(await read("Age")).toBe("2026-11-21T14:02:58.000Z");

  await vi.advanceTimersByTimeAsync(1000);
  expect([await read("Code"), await read("Age")]).toEqual([
    "2026-11-21T14:02:59.000Z",
    "2026-11-21T14:02:58.000Z",
  ]);
  await vi.advanceTimersByTimeAsync(1000);
  expect(await read("Age")).toBe("2026-11-21T14:03:00.000Z");
  expect(vi.getTimerCount()).toBe(1);
});

test("no timer runs while the tab is hidden, and turning visible reads now at once", async () => {
  useFakeClock("2026-11-21T14:02:30.000Z");
  onTestFinished(() => {
    Reflect.deleteProperty(document, "visibilityState");
  });
  renderKit(<Reader name="Age" />);
  expect(await read("Age")).toBe("2026-11-21T14:02:30.000Z");

  setVisibility("hidden");
  expect(vi.getTimerCount()).toBe(0);
  vi.setSystemTime(new Date("2026-11-21T14:09:10.000Z"));
  setVisibility("visible");
  expect(await read("Age")).toBe("2026-11-21T14:09:10.000Z");
  expect(vi.getTimerCount()).toBe(1);
});

test("with nothing reading the clock, the first reader reads the current now", async () => {
  useFakeClock("2026-11-07T15:00:00.000Z");
  renderKit(<Reader name="Age" />);
  expect(await read("Age")).toBe("2026-11-07T15:00:00.000Z");
});
