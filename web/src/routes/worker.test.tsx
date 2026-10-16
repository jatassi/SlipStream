import { http } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { getGetStatusMockHandler } from "../api/gen/query/system/system.msw";
import { problem, signedOut } from "../test/handlers";
import { renderApp } from "../test/render-app";

const workerNode = [
  getGetStatusMockHandler({ claim: "worker" }),
  http.get("*/api/v1/session", () =>
    problem(409, "worker_node", "This Node is a Worker. Manage it from its Controller."),
  ),
];

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
      "This Node is a Worker. Manage it from its Controller.",
    );
  },
);

test("/worker on a Controller with no session goes to sign-in", async () => {
  const { router } = renderApp({ path: "/worker", handlers: [signedOut] });
  await expect.element(page.getByRole("heading", { name: "Sign in to SlipStream" })).toBeVisible();
  expect(router.state.location.href).toBe("/sign-in");
});
