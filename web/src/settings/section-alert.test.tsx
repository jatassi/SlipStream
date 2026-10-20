import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { fillingVolume, fullVolume, healthServer, reportOf } from "../test/health";
import { serveRoots } from "../test/roots";
import { openRoots } from "../test/roots-page";
import { renderApp } from "../test/render-app";
import { asPhone } from "../test/viewport";

const settingsTabs = () => page.getByRole("navigation", { name: "Settings" });

test("the Library roots tab carries a red dot with its counts in words while Storage has an error", async () => {
  serveRoots();
  const health = healthServer(reportOf({ storage: [fullVolume, fillingVolume] }));
  renderApp({ path: "/settings/address", handlers: health.handlers });
  await expect
    .element(settingsTabs().getByRole("link", { name: "Library roots 1 error, 1 warning" }))
    .toBeVisible();
  await expect.element(settingsTabs().getByRole("link", { name: "Address" })).toBeVisible();
});

test("Library roots opens with its worst Health item, called an error, leading to it on the Health page", async () => {
  serveRoots();
  const health = healthServer(reportOf({ storage: [fillingVolume, fullVolume] }));
  const { router } = openRoots({ handlers: health.handlers });
  const line = page.getByText(fullVolume.text);
  await expect.element(line).toBeVisible();
  await expect.element(page.getByText("Error", { exact: true }).first()).toBeVisible();
  await expect.element(page.getByText(fillingVolume.text)).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Open Health" }));
  await expect.poll(() => router.state.location.pathname).toBe("/health");
  expect(router.state.location.hash).toBe(`health-storage-${fullVolume.key}`);
});

test("with nothing wrong in Storage Library roots has no dot and no line at its top", async () => {
  serveRoots();
  openRoots({ handlers: healthServer(reportOf()).handlers });
  await expect
    .element(settingsTabs().getByRole("link", { name: "Library roots", exact: true }))
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
