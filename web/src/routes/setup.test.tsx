import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { readPref, writePref } from "../shell/prefs";
import { renderApp } from "../test/render-app";
import {
  forgetKeptCode,
  joinExisting,
  rightCode,
  setupCode,
  startNew,
  unclaimedNode,
} from "../test/setup-node";

forgetKeptCode();

test("the Setup code shows grouped as it is typed", async () => {
  renderApp({ path: "/setup", handlers: unclaimedNode().handlers });
  await userEvent.type(setupCode(), "7k2m q9xd 4hbt");
  await expect.element(setupCode()).toHaveValue("7K2M-Q9XD-4HBT");
});

test("a kept code the Node no longer takes says the Setup code changed", async () => {
  writePref("setupCode", "AAAABBBBCCCC");
  renderApp({ path: "/setup", handlers: unclaimedNode().handlers });
  await expect
    .element(
      page.getByText(
        "This Node restarted, so its Setup code changed. Read the new one from its log.",
      ),
    )
    .toBeVisible();
  expect(readPref("setupCode")).toBe("");
});

test("Back returns to the choice before anything is created", async () => {
  writePref("setupCode", rightCode);
  renderApp({ path: "/setup", handlers: unclaimedNode().handlers });
  await userEvent.click(startNew());
  await expect.element(page.getByRole("heading", { name: "Start a new SlipStream" })).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Back" }));
  await expect.element(joinExisting()).toBeVisible();
});
