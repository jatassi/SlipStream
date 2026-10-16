import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getGetSystemMockHandler } from "../api/gen/query/system/system.msw";
import { problem, signedInAsGrace } from "../test/handlers";
import { renderApp } from "../test/render-app";

const mainNav = () => page.getByRole("navigation", { name: "Main" });

const system = getGetSystemMockHandler({
  build: { stamped: true, version: "0.4.0" },
  dataDir: "/data/home",
  logsDir: "/data/home/logs",
  http: { configured: "127.0.0.2:8080" },
  listeners: {
    http: "127.0.0.2:8080",
    torrent: "127.0.0.2:6881",
    link: "127.0.0.2:8485",
    control: "/data/home/control.sock",
  },
  pid: 4242,
  seed: "home",
  startedAt: "2026-10-12T09:00:00Z",
});

test("a page that fails says why in the content, keeps the sidebar, and Try again reloads it", async () => {
  let failing = true;
  renderApp({
    path: "/system",
    handlers: [
      http.get("*/api/v1/system", () =>
        failing ? problem(500, "internal_error", "Something went wrong on this Node") : undefined,
      ),
      system,
    ],
  });
  const alert = page.getByRole("alert");
  await expect.element(alert).toHaveTextContent("Something went wrong on this Node");
  await expect.element(mainNav().getByRole("link", { name: "Activity" })).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect.element(page.getByText("internal_error")).toBeVisible();
  await expect.element(page.getByText("500", { exact: true })).toBeVisible();

  failing = false;
  await userEvent.click(page.getByRole("button", { name: "Try again" }));
  await expect.element(page.getByRole("heading", { name: "This Node" })).toBeVisible();
  await expect.element(page.getByText("0.4.0")).toBeVisible();
  await expect.element(alert).not.toBeInTheDocument();
});

test("a page whose server doesn't answer says so", async () => {
  renderApp({
    path: "/system",
    handlers: [http.get("*/api/v1/system", () => HttpResponse.error())],
  });
  await expect
    .element(page.getByRole("alert"), { timeout: 8000 })
    .toHaveTextContent("SlipStream isn't answering. Check that it's running, then try again.");
  await expect.element(page.getByRole("button", { name: "Try again" })).toBeVisible();
  await expect.element(mainNav()).toBeVisible();
});

test("a page that throws in the browser says it couldn't load and shows the message", async () => {
  renderApp({
    path: "/system",
    handlers: [http.get("*/api/v1/system", () => HttpResponse.json({ build: {} }))],
  });
  await expect.element(page.getByRole("alert")).toHaveTextContent("This page couldn't load.");
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect.element(page.getByText("Invalid time value")).toBeVisible();
  await expect.element(mainNav()).toBeVisible();
});

test("a page for Admins says so and links to the portal", async () => {
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

test("a dead address in the app says there's nothing there and links to Library", async () => {
  renderApp({ path: "/nowhere/at/all" });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
  await expect
    .element(page.getByRole("button", { name: "Go to Library" }))
    .toHaveAttribute("href", "/library");
  await expect.element(mainNav()).toBeVisible();
});

test("a dead address in the portal links a Member home", async () => {
  renderApp({ path: "/portal/nowhere", handlers: [signedInAsGrace] });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
  await expect
    .element(page.getByRole("button", { name: "Go to Home" }))
    .toHaveAttribute("href", "/portal");
  await expect.element(page.getByRole("navigation", { name: "Portal" })).toBeVisible();
});

test("when the shell can't start, the card takes the screen with Reload", async () => {
  renderApp({
    path: "/activity",
    handlers: [
      http.get("*/api/v1/session", () => problem(500, "internal_error", "The session store broke")),
    ],
  });
  await expect.element(page.getByRole("alert")).toHaveTextContent("The session store broke");
  await expect.element(page.getByRole("button", { name: "Reload" })).toBeVisible();
  await expect.element(mainNav()).not.toBeInTheDocument();
});
