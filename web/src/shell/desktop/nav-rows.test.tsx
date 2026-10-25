import { HttpResponse, http } from "msw";
import { expect, onTestFinished, test } from "vitest";
import { commands, page } from "vitest/browser";

import type { DownloadclientsAggregate } from "../../api/gen/model";
import { liveServer } from "../../test/live-server";
import { renderApp } from "../../test/render-app";
import { asPhone } from "../../test/viewport";

const mainNav = () => page.getByRole("navigation", { name: "Main" });
const tabs = () => page.getByRole("navigation", { name: "Tabs" });

const downloading = (downloads: DownloadclientsAggregate) =>
  http.get("*/api/v1/counts", () =>
    HttpResponse.json({ activity: 2, downloads, alerts: { nodes: null } }),
  );

const reached = (bar: Element) => {
  const indicator = bar.firstElementChild;
  if (indicator === null) {
    throw new Error("the fill draws no indicator");
  }
  return indicator.getBoundingClientRect().width / bar.getBoundingClientRect().width;
};

const shade = (bar: Element) => {
  const indicator = bar.firstElementChild;
  if (indicator === null) {
    throw new Error("the fill draws no indicator");
  }
  return Number(getComputedStyle(indicator).opacity);
};

const live = (downloads: DownloadclientsAggregate) => ({
  event: "torrents.live" as const,
  data: { at: new Date().toISOString(), rows: [], downloads },
});

test("the Activity row fills with the downloads' combined progress and follows torrents.live", async () => {
  const server = liveServer();
  renderApp({
    path: "/library",
    handlers: [...server.handlers, downloading({ done: 30, wanted: 120, allPaused: false })],
  });
  const fill = mainNav().getByRole("progressbar", { name: "Downloads" });
  await expect.element(fill).toHaveAttribute("aria-valuetext", "25%");
  await expect.element(mainNav().getByRole("link", { name: "Activity 2 live" })).toBeVisible();
  await expect.poll(() => reached(fill.element())).toBeCloseTo(0.25, 2);
  expect(shade(fill.element())).toBe(1);

  server.clients[0]?.send(live({ done: 90, wanted: 120, allPaused: false }));
  await expect.element(fill).toHaveAttribute("aria-valuetext", "75%");
  await expect.poll(() => reached(fill.element())).toBeCloseTo(0.75, 2);
});

test("the fill dims when every download is paused, and brightens once one resumes", async () => {
  const server = liveServer();
  renderApp({
    path: "/library",
    handlers: [...server.handlers, downloading({ done: 30, wanted: 120, allPaused: true })],
  });
  const fill = mainNav().getByRole("progressbar", { name: "Downloads" });
  await expect.element(fill).toHaveAttribute("aria-valuetext", "25%, paused");
  await expect.poll(() => shade(fill.element())).toBeCloseTo(0.4, 2);

  server.clients[0]?.send(live({ done: 30, wanted: 120, allPaused: false }));
  await expect.element(fill).toHaveAttribute("aria-valuetext", "25%");
  await expect.poll(() => shade(fill.element())).toBe(1);
});

test("with nothing to download the Activity row draws no fill", async () => {
  const server = liveServer();
  renderApp({
    path: "/library",
    handlers: [...server.handlers, downloading({ done: 30, wanted: 120, allPaused: false })],
  });
  const fill = mainNav().getByRole("progressbar", { name: "Downloads" });
  await expect.element(fill).toBeInTheDocument();

  server.clients[0]?.send(live({ done: 0, wanted: 0, allPaused: false }));
  await expect.element(fill).not.toBeInTheDocument();
});

test("under reduced motion the fill jumps to each sample", async () => {
  await commands.emulateReducedMotion(true);
  onTestFinished(() => commands.emulateReducedMotion(false));
  const server = liveServer();
  renderApp({
    path: "/library",
    handlers: [...server.handlers, downloading({ done: 30, wanted: 120, allPaused: false })],
  });
  const fill = mainNav().getByRole("progressbar", { name: "Downloads" });
  await expect.element(fill).toHaveAttribute("aria-valuetext", "25%");

  server.clients[0]?.send(live({ done: 90, wanted: 120, allPaused: true }));
  await expect.element(fill).toHaveAttribute("aria-valuetext", "75%, paused");
  expect(reached(fill.element())).toBeCloseTo(0.75, 2);
  expect(shade(fill.element())).toBeCloseTo(0.4, 2);
});

test("on a phone the Activity tab carries the fill", async () => {
  await asPhone();
  renderApp({
    path: "/library",
    handlers: [downloading({ done: 60, wanted: 120, allPaused: true })],
  });
  const fill = tabs().getByRole("progressbar", { name: "Downloads" });
  await expect.element(fill).toHaveAttribute("aria-valuetext", "50%, paused");
  await expect.poll(() => reached(fill.element())).toBeCloseTo(0.5, 2);
  await expect
    .poll(() => tabs().getByRole("listitem").nth(1).element().textContent)
    .toBe("Activity2");
});
