import { http } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { getGetSystemMockHandler } from "../../api/gen/query/system/system.msw";
import { problem } from "../../test/handlers";
import { renderApp } from "../../test/render-app";

test("/system shows this Node's build and data dir", async () => {
  renderApp({
    path: "/system",
    handlers: [
      getGetSystemMockHandler({
        build: { stamped: true, version: "0.0.0-dev+3f2a1c9" },
        dataDir: "/data/home",
        listeners: {
          http: "127.0.0.2:8080",
          torrent: "127.0.0.2:6881",
          link: "127.0.0.2:8485",
          control: "/data/home/control.sock",
        },
        pid: 4242,
        seed: "home",
        startedAt: "2026-10-12T09:00:00Z",
      }),
    ],
  });
  await expect.element(page.getByRole("heading", { name: "This Node" })).toBeVisible();
  await expect.element(page.getByText("0.0.0-dev+3f2a1c9")).toBeVisible();
  await expect.element(page.getByText("/data/home", { exact: true })).toBeVisible();
});

test("a Member gets the server's refusal", async () => {
  renderApp({
    path: "/system",
    handlers: [
      http.get("*/api/v1/system", () => problem(403, "needs_admin", "Only an Admin can do this")),
    ],
  });
  await expect.element(page.getByRole("alert")).toHaveTextContent("This page is for Admins.");
  await expect
    .element(page.getByRole("button", { name: "Open the portal" }))
    .toHaveAttribute("href", "/portal");
});
