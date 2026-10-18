import { HttpResponse, http } from "msw";
import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { liveServer } from "../../test/live-server";
import { nodeSystems, restarts } from "../../test/node-system";
import { homeId, homeSystem } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { worker } from "../../test/worker";
import { thisPage } from "../reload";

test("restarting SlipStream waits quietly through failed polls and reloads once a later start answers", async () => {
  const reload = vi.spyOn(thisPage, "reload").mockImplementation(() => {});
  const server = liveServer();
  const restart = restarts();
  let polls = 0;
  const app = renderApp({
    path: `/nodes/${homeId}/web`,
    handlers: [...server.handlers, ...nodeSystems().handlers, restart.handler],
  });
  await expect.poll(() => app.live.getState()).toBe("open");
  await userEvent.click(page.getByRole("button", { name: /^Account/u }));
  await userEvent.click(page.getByRole("menuitem", { name: "Restart SlipStream…" }));
  worker.use(
    http.get("*/api/v1/system", () => {
      polls += 1;
      return polls <= 3
        ? HttpResponse.error()
        : HttpResponse.json({ ...homeSystem, startedAt: "2026-10-12T09:05:00Z" });
    }),
  );
  await userEvent.click(page.getByRole("button", { name: "Restart", exact: true }));

  await expect.element(page.getByText("Restarting SlipStream…")).toBeVisible();
  expect(restart.asked).toEqual([homeId]);

  worker.use(http.get("*/api/v1/events", () => HttpResponse.error()));
  server.clients[0]?.error();
  await expect.poll(() => app.live.getState()).toBe("reconnecting");
  await expect.element(page.getByText("Reconnecting to SlipStream…")).not.toBeInTheDocument();

  await expect.poll(() => polls, { timeout: 5000 }).toBe(3);
  expect(reload).not.toHaveBeenCalled();
  await expect.poll(() => reload.mock.calls.length, { timeout: 3000 }).toBe(1);
});
