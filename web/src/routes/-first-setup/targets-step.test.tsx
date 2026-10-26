import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../../api/gen/query/works/works.msw";
import { firstSetup, stepHeading, todo } from "../../test/first-setup";
import { changeOp, noImpact } from "../../test/impact";
import { cinema } from "../../test/library";
import { films, profileHandlers, ultra } from "../../test/profiles";
import { renderApp } from "../../test/render-app";
import { zines } from "../../test/roots-page";
import { starter, targetHandlers } from "../../test/targets";

function open() {
  const server = firstSetup([todo("targets")]);
  const saving = changeOp("put", `*/api/v1/targets/${starter.id}`, () => noImpact, {
    target: starter,
  });
  renderApp({
    path: "/setup/targets",
    handlers: [
      ...server.handlers,
      saving.handler,
      ...targetHandlers(),
      ...profileHandlers([films, ultra]),
      getListContentTypesMockHandler({ contentTypes: [cinema, { ...zines, enabled: false }] }),
    ],
  });
  return { server, saving };
}

const filmsRow = () => page.getByRole("listitem", { name: "Films" });

test("the Targets step says in one sentence what each enabled type grabs, from its default Profile", async () => {
  open();
  await expect.element(stepHeading("What SlipStream grabs")).toBeVisible();
  await expect
    .element(filmsRow().getByText(/^Grabs /u))
    .toHaveTextContent(
      "Grabs HDTV-720p, WEBRip-720p or WEBDL-720p or better. Upgrades until Bluray-1080p or Remux-1080p, then stops.",
    );
  await expect.element(page.getByRole("listitem", { name: "Zines" })).not.toBeInTheDocument();
});

test("Details holds the Profile, qualities, cutoff and sizes behind the sentence", async () => {
  open();
  await userEvent.click(filmsRow().getByRole("button", { name: "Details" }));
  const shown = filmsRow().getByRole("definition");
  await expect.element(shown.nth(0)).toHaveTextContent("Films");
  await expect
    .element(shown.nth(1))
    .toHaveTextContent(
      "Bluray-1080p, Remux-1080p, HDTV-1080p, WEBRip-1080p, WEBDL-1080p, Bluray-720p, HDTV-720p, WEBRip-720p, WEBDL-720p",
    );
  await expect.element(shown.nth(2)).toHaveTextContent("Bluray-1080p");
  await expect
    .element(shown.nth(3).getByText("WEBDL-1080p, 12.5 to 40 MiB per minute"))
    .toBeVisible();
});

test("Edit opens the Target editor in place, and saving closes it on the same step", async () => {
  const { server, saving } = open();
  await userEvent.click(filmsRow().getByRole("button", { name: "Edit 1080p" }));
  const editor = page.getByRole("dialog", { name: "Edit 1080p" });
  await expect.element(editor.getByRole("textbox", { name: "Name" })).toHaveValue("1080p");
  await userEvent.click(editor.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved 1080p.")).toBeVisible();
  await expect.element(editor).not.toBeInTheDocument();
  await expect.element(stepHeading("What SlipStream grabs")).toBeVisible();
  expect(saving.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "0" },
  ]);
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("First setup")).toBeVisible();
  expect(server.writes).toEqual([{ id: "targets", outcome: "done" }]);
});
