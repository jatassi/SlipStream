import { useEffect } from "react";
import { beforeEach, expect, onTestFinished, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { frames } from "../test/frames";
import { renderKit } from "../test/render-kit";
import { toast, toastManager } from "./toast";
import { Toasts } from "./toasts";

beforeEach(() => {
  onTestFinished(() => {
    toastManager.close();
  });
});

function useFakeClock() {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date"] });
  onTestFinished(() => {
    vi.useRealTimers();
  });
}

const notifications = () => page.getByRole("region", { name: "Notifications" });

// The manager drops a toast nobody listens to, and the viewport's Provider
// subscribes in an effect, which runs before a later sibling's effect.
async function mountToasts(shell: "desktop" | "phone" = "desktop") {
  await new Promise<void>((resolve) => {
    function Subscribed() {
      useEffect(resolve, []);
      return null;
    }
    renderKit(
      <>
        <Toasts />
        <Subscribed />
      </>,
      { shell },
    );
  });
}

// Polling advances a fake clock, so timed checks wait real frames for any exit
// animation to finish and then look once.
const settle = () => frames(30);

const shown = (message: string) => notifications().getByText(message).query() !== null;

test("a toast announces politely and is gone after 2.8s", async () => {
  await mountToasts();
  await expect.element(notifications()).toHaveAttribute("aria-live", "polite");
  useFakeClock();
  toast("Search started");

  await vi.advanceTimersByTimeAsync(2799);
  await settle();
  expect(shown("Search started")).toBe(true);
  await vi.advanceTimersByTimeAsync(1);
  await settle();
  expect(shown("Search started")).toBe(false);
});

test("a toast with an action stays until it is acted on", async () => {
  await mountToasts();
  useFakeClock();
  const onClick = vi.fn<() => void>();
  toast("Removed 3 torrents", { action: { label: "Undo", onClick } });

  await vi.advanceTimersByTimeAsync(60_000);
  await settle();
  expect(shown("Removed 3 torrents")).toBe(true);
  await userEvent.click(notifications().getByRole("button", { name: "Undo" }));
  expect(onClick).toHaveBeenCalledOnce();
  await settle();
  expect(shown("Removed 3 torrents")).toBe(false);
});

test("a toast with an action can be dismissed", async () => {
  await mountToasts();
  const onClick = vi.fn<() => void>();
  toast("Removed 3 torrents", { action: { label: "Undo", onClick } });
  await userEvent.click(notifications().getByRole("button", { name: "Dismiss" }));
  await expect.element(notifications().getByText("Removed 3 torrents")).not.toBeInTheDocument();
  expect(onClick).not.toHaveBeenCalled();
});

test("a toast stays while the pointer rests on it", async () => {
  await mountToasts();
  toast("Search started");
  await userEvent.hover(notifications().getByText("Search started"));
  useFakeClock();
  await vi.advanceTimersByTimeAsync(10_000);
  await settle();
  expect(shown("Search started")).toBe(true);
});

test("a fail toast carries the fail word", async () => {
  await mountToasts();
  toast("TMDB did not answer", { kind: "fail" });
  const capsule = notifications().getByRole("dialog", { name: "TMDB did not answer" });
  await expect.element(capsule.getByText("Failed", { exact: true })).toBeVisible();
});

test("three toasts show at once, newest on top", async () => {
  await mountToasts();
  for (const message of ["First", "Second", "Third", "Fourth"]) {
    toast(message);
  }
  await expect.element(notifications().getByText("Fourth")).toBeVisible();
  await expect.element(notifications().getByText("First")).not.toBeVisible();
  const top = (name: string) =>
    notifications().getByText(name).element().getBoundingClientRect().top;
  expect(top("Fourth")).toBeLessThan(top("Third"));
  expect(top("Third")).toBeLessThan(top("Second"));
});

test("toasts sit top right in the desktop shell", async () => {
  await mountToasts();
  toast("Search started");
  const capsule = notifications().getByRole("dialog", { name: "Search started" });
  await expect.element(capsule).toBeVisible();
  const { right } = capsule.element().getBoundingClientRect();
  expect(document.documentElement.clientWidth - right).toBe(28);
});

test("toasts sit top centre in the phone shell", async () => {
  await mountToasts("phone");
  toast("Search started");
  const capsule = notifications().getByRole("dialog", { name: "Search started" });
  await expect.element(capsule).toBeVisible();
  const { left, width } = capsule.element().getBoundingClientRect();
  expect(left + width / 2).toBeCloseTo(document.documentElement.clientWidth / 2, 0);
});
