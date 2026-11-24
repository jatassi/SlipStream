import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { fillingVolume, healthServer, reportOf } from "../test/health";
import { serveRoots } from "../test/roots";
import { openRoots } from "../test/roots-page";
import { renderApp } from "../test/render-app";
import { asPhone } from "../test/viewport";

const settingsNav = () => page.getByRole("navigation", { name: "Settings" });

test("with nothing wrong in Storage Library roots has no dot and no line at its top", async () => {
  serveRoots();
  openRoots({ handlers: healthServer(reportOf()).handlers });
  await expect
    .element(settingsNav().getByRole("link", { name: "Library roots", exact: true }))
    .toBeVisible();
  expect(page.getByRole("button", { name: "Open Health" }).query()).toBeNull();
});

test("on a phone the Settings list row carries the same dot and words", async () => {
  await asPhone();
  serveRoots();
  renderApp({
    path: "/settings",
    handlers: healthServer(reportOf({ storage: [fillingVolume] })).handlers,
  });
  await expect.element(page.getByRole("link", { name: "Library roots 1 warning" })).toBeVisible();
  await expect.element(page.getByRole("link", { name: "Address", exact: true })).toBeVisible();
});
