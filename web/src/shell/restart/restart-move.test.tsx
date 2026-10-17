import { http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { nodeSystems, restarts } from "../../test/node-system";
import { homeId } from "../../test/nodes";
import { renderApp } from "../../test/render-app";

test("a restart onto a new port links to it when the browser reached the Node on its port, and doesn't poll", async () => {
  const restart = restarts({ boundPort: Number(location.port), nextPort: 9000 });
  let polls = 0;
  renderApp({
    path: `/nodes/${homeId}`,
    handlers: [
      ...nodeSystems().handlers,
      restart.handler,
      http.get("*/api/v1/system", () => {
        polls += restart.asked.length;
      }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Restart", exact: true }));
  await userEvent.click(page.getByRole("dialog").getByRole("button", { name: "Restart" }));

  await expect.element(page.getByText("SlipStream is moving to port 9000")).toBeVisible();
  const next = new URL(location.href);
  next.port = "9000";
  await expect
    .element(page.getByRole("button", { name: "Open SlipStream on port 9000" }))
    .toHaveAttribute("href", next.href);
  expect(polls).toBe(0);
});
