import { HttpResponse, http } from "msw";
import { expect, onTestFinished, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { nodeSystems, restarts } from "../../test/node-system";
import { homeId } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { failAfter } from "./restart-state";

test("a restart with no answer for 90 seconds shows the failure card with the log folder, and Try again waits again", async () => {
  const restart = restarts();
  renderApp({
    path: `/nodes/${homeId}`,
    handlers: [
      ...nodeSystems().handlers,
      restart.handler,
      http.get("*/api/v1/system", () =>
        restart.asked.length === 0 ? undefined : HttpResponse.error(),
      ),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Restart", exact: true }));
  await userEvent.click(page.getByRole("dialog").getByRole("button", { name: "Restart" }));
  await expect.element(page.getByText("Restarting SlipStream…")).toBeVisible();

  onTestFinished(() => {
    vi.useRealTimers();
  });
  vi.setSystemTime(Date.now() + failAfter);
  await expect.element(page.getByRole("alert")).toHaveTextContent("SlipStream hasn't come back.");
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect.element(page.getByText("/data/home/logs")).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Try again" }));
  await expect.element(page.getByText("Restarting SlipStream…")).toBeVisible();
  await expect.element(page.getByRole("alert")).not.toBeInTheDocument();
});
