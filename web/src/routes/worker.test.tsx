import { http } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { ServingWorker } from "../api/gen/model";
import { getGetServingMockHandler } from "../api/gen/query/serving/serving.msw";
import { getGetStatusMockHandler } from "../api/gen/query/system/system.msw";
import { noPasskeys, problem, signedOut } from "../test/handlers";
import { renderApp } from "../test/render-app";
import { worker } from "../test/worker";

const seedbox: ServingWorker = {
  name: "seedbox",
  controller: "home",
  fingerprint: "7Q2M-0K4D-91XB-3FHA",
  holding: ["Searching", "Downloading"],
  seeding: 3,
};

const workerNode = [
  getGetStatusMockHandler({ claim: "worker", passkeys: noPasskeys }),
  getGetServingMockHandler(seedbox),
  http.get("*/api/v1/session", () =>
    problem(409, "worker_node", "This Node is a Worker. Manage it from its Controller."),
  ),
];

const unreachable = () => problem(503, "controller_unreachable", "The Controller can't be reached");

test.each(["/", "/nodes", "/sign-in", "/setup"])(
  "a visit to %s on a Worker lands on /worker",
  async (path) => {
    const { router } = renderApp({ path, handlers: workerNode });
    await expect
      .element(
        page.getByRole("heading", {
          name: "This Node is a Worker. Manage it from its Controller.",
        }),
      )
      .toBeVisible();
    expect(router.state.location.href).toBe("/worker");
    expect(document.querySelector("main")?.textContent).toBe(
      "This Node is a Worker. Manage it from its Controller.NameseedboxControllerhomeKey7Q2M-0K4D-91XB-3FHA",
    );
  },
);

test.each(["/", "/movies", "/sign-in", "/worker"])(
  "a visit to %s at a Serving Worker that can't reach its Controller lands on its status card",
  async (path) => {
    const since = new Date(Date.now() - 60_000).toISOString();
    const { router } = renderApp({
      path,
      handlers: [
        getGetServingMockHandler({ ...seedbox, since }),
        http.get("*/api/v1/session", unreachable),
        http.get("*/api/v1/status", unreachable),
      ],
    });
    await expect
      .element(
        page.getByRole("heading", { name: /^home can't be reached from seedbox since .+\.$/u }),
      )
      .toBeVisible();
    expect(router.state.location.href).toBe("/serving");
    await expect
      .element(page.getByText("seedbox keeps seeding and finishes what it started."))
      .toBeVisible();
    await expect
      .element(page.getByText("Holding Searching and Downloading · seeding 3 torrents."))
      .toBeVisible();
  },
);

test("the status card counts one torrent and names no Role when it holds none", async () => {
  renderApp({
    path: "/serving",
    handlers: [
      getGetServingMockHandler({
        ...seedbox,
        since: new Date().toISOString(),
        holding: [],
        seeding: 1,
      }),
      http.get("*/api/v1/session", unreachable),
    ],
  });
  await expect.element(page.getByText("Seeding 1 torrent.")).toBeVisible();
});

test("the status card goes back to the app once the Controller answers", async () => {
  const { router } = renderApp({
    path: "/serving",
    handlers: [
      getGetServingMockHandler({ ...seedbox, since: new Date().toISOString() }),
      http.get("*/api/v1/session", unreachable),
      http.get("*/api/v1/status", unreachable),
    ],
  });
  await expect.element(page.getByRole("heading", { name: /can't be reached/u })).toBeVisible();
  worker.use(signedOut, getGetStatusMockHandler({ claim: "controller", passkeys: noPasskeys }));
  await expect
    .element(page.getByRole("heading", { name: "Sign in to SlipStream" }), { timeout: 10_000 })
    .toBeVisible();
  expect(router.state.location.href).toBe("/sign-in?redirect=%2F");
});

test("/worker on a Controller with no session goes to sign-in", async () => {
  const { router } = renderApp({ path: "/worker", handlers: [signedOut] });
  await expect.element(page.getByRole("heading", { name: "Sign in to SlipStream" })).toBeVisible();
  expect(router.state.location.href).toBe("/sign-in");
});
