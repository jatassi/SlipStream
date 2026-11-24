import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { homeId } from "../../../test/nodes";
import { homeNetwork, networkServer } from "../../../test/remote-access";
import { renderApp } from "../../../test/render-app";

test("a Node's Network section turns on Open router ports automatically for that Node", async () => {
  const server = networkServer();
  renderApp({ path: `/nodes/${homeId}/network`, handlers: server.handlers });
  await expect
    .element(page.getByRole("navigation", { name: "home" }).getByRole("link", { name: "Network" }))
    .toHaveAttribute("aria-current", "page");
  await expect.element(page.getByText(homeNetwork.line)).toBeVisible();
  await userEvent.click(page.getByRole("switch", { name: "Open router ports automatically" }));
  await expect.poll(() => server.sent).toEqual([{ values: { "reachability.mapPorts": true } }]);
});

test("an offline Node's Network section says its network is from when it was last online", async () => {
  renderApp({
    path: `/nodes/${homeId}/network`,
    handlers: networkServer({ ...homeNetwork, now: false }).handlers,
  });
  await expect.element(page.getByText(/^home is offline\./u)).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect.element(page.getByText("192.168.1.20")).toBeVisible();
});
