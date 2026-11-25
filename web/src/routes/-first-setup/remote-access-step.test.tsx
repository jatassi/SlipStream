import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { done, firstSetup, stepHeading, todo } from "../../test/first-setup";
import {
  livingRoomPlexId,
  nothingSetUp,
  plexSuggestion,
  remoteAccessServer,
  uiSuggestion,
} from "../../test/remote-access";
import { renderApp } from "../../test/render-app";

function open(suggestions = [uiSuggestion, plexSuggestion]) {
  const setup = firstSetup([done("targets"), todo("remote-access")]);
  const remote = remoteAccessServer({ ...nothingSetUp, suggestions });
  renderApp({ path: "/setup/remote-access", handlers: [...setup.handlers, ...remote.handlers] });
  return { setup, remote };
}

const continueButton = () => page.getByRole("button", { name: "Continue" });

test("Continue sets up each switched-on suggestion, checks them, then marks the step done", async () => {
  const { setup, remote } = open();
  await userEvent.click(page.getByRole("switch", { name: uiSuggestion.subject }));
  await userEvent.click(continueButton());
  await expect.element(stepHeading("First setup")).toBeVisible();
  expect(remote.sent).toEqual([
    {
      what: "add",
      body: { kind: "media-server", serverId: livingRoomPlexId, public: plexSuggestion.public },
    },
    { what: "run" },
  ]);
  expect(setup.writes).toEqual([{ id: "remote-access", outcome: "done" }]);
});

test("a Media Server with no public address asks for one and sets up nothing", async () => {
  const { setup, remote } = open([{ ...plexSuggestion, public: "" }]);
  await userEvent.click(continueButton());
  await expect
    .element(page.getByRole("textbox", { name: "Public address" }))
    .toHaveAccessibleDescription(/Enter the address/u);
  expect(remote.sent).toEqual([]);
  expect(setup.writes).toEqual([]);
});
