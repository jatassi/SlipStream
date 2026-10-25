import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { UpdateProfileBody } from "../api/gen/zod/targets/targets.zod";
import { changeOp, noImpact } from "../test/impact";
import { cinema } from "../test/library";
import { profileHandlers, ultra } from "../test/profiles";
import { renderApp } from "../test/render-app";
import { starter, targetHandlers } from "../test/targets";

const handlers = () => [
  ...targetHandlers(),
  ...profileHandlers(),
  getListContentTypesMockHandler({ contentTypes: [cinema] }),
];

test("saving a Profile's Targets shows the Works that keep their own Edition list, and sends the edit", async () => {
  const op = changeOp(
    "put",
    `*/api/v1/profiles/${ultra.id}`,
    () => ({ ...noImpact, works: 3, ownList: 1, renamed: 2 }),
    { profile: ultra },
  );
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: [op.handler, ...handlers()],
  });
  await userEvent.click(page.getByRole("button", { name: "Remove 2160p HDR" }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  const sheet = page.getByRole("dialog", { name: "Save Ultra" });
  await expect
    .element(sheet.getByRole("list", { name: "What changes" }))
    .toHaveTextContent(
      "3 Works change.1 Work keeps its own Edition list, so the new list doesn't apply to it.2 files are renamed.",
    );
  expect(sheet.getByRole("radiogroup").query()).toBeNull();
  await userEvent.click(sheet.getByRole("button", { name: "Save", exact: true }));

  await expect.element(page.getByText("Saved Ultra.")).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "0" },
  ]);
  expect(op.sent[1]?.body).toEqual({
    name: "Ultra",
    contentType: "cinema",
    targets: [{ targetId: starter.id, sourcing: "download" }],
    editions: { kind: "any" },
  });
});

test("saving a Profile's chosen cuts previews them through the Impact, then sends them", async () => {
  const op = changeOp(
    "put",
    `*/api/v1/profiles/${ultra.id}`,
    () => ({ ...noImpact, works: 1, newlyWanted: 2 }),
    { profile: ultra },
  );
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: [op.handler, ...handlers()],
  });
  await userEvent.click(page.getByRole("radio", { name: "Choose cuts" }));
  await userEvent.click(page.getByRole("button", { name: "Add cut" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Superfan" }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  const sheet = page.getByRole("dialog", { name: "Save Ultra" });
  await expect
    .element(sheet.getByRole("list", { name: "What changes" }))
    .toHaveTextContent("1 Work changes.2 cells become wanted.");
  await userEvent.click(sheet.getByRole("button", { name: "Save", exact: true }));

  await expect.element(page.getByText("Saved Ultra.")).toBeVisible();
  const editions = { kind: "cuts", cuts: ["standard", "superfan"] };
  expect(op.sent.map((call) => [call.query, UpdateProfileBody.parse(call.body).editions])).toEqual([
    [{ preview: "true" }, editions],
    [{ unwanted: "delete", confirm: "0" }, editions],
  ]);
});
