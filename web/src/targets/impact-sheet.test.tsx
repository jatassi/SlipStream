import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TargetsImpact } from "../api/gen/model";
import { changeOp, noImpact } from "../test/impact";
import { renderApp } from "../test/render-app";
import { starter, targetHandlers } from "../test/targets";

const oneUnwanted: TargetsImpact = {
  works: 2,
  ownList: 0,
  unwanted: { files: 1, bytes: 734_003_200 },
  renamed: 3,
  newlyWanted: 4,
  downloading: 1,
};

const saving = (answer: Parameters<typeof changeOp>[2]) =>
  changeOp("put", `*/api/v1/targets/${starter.id}`, answer, { target: starter });

const sheet = () => page.getByRole("dialog", { name: "Save 1080p" });
const lines = () =>
  sheet()
    .getByRole("list", { name: "What changes" })
    .getByRole("listitem")
    .all()
    .map((line) => line.element().textContent);

async function editAndSave() {
  await userEvent.click(page.getByRole("checkbox", { name: "WEBDL-1080p" }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
}

test("saving a Target shows its Impact in plain counts, with Delete chosen and the count on the button", async () => {
  const op = saving(() => oneUnwanted);
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [op.handler, ...targetHandlers()],
  });
  await editAndSave();

  await expect.element(sheet()).toBeVisible();
  expect(lines()).toEqual([
    "2 Works change.",
    "1 file would fill nothing (700 MB).",
    "3 files are renamed.",
    "4 cells become wanted.",
    "1 download in progress is for a cell that is no longer wanted. Its file is deleted or kept, as chosen below.",
  ]);
  await expect.element(sheet().getByRole("radio", { name: "Delete" })).toBeChecked();
  await expect
    .element(sheet().getByRole("button", { name: "Save and delete 1 file" }))
    .toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([{ preview: "true" }]);

  await userEvent.click(sheet().getByRole("radio", { name: "Keep as Kept files" }));
  await userEvent.click(sheet().getByRole("button", { name: "Save and keep 1 file" }));

  await expect.element(page.getByText("Saved 1080p.")).toBeVisible();
  await expect.element(page.getByRole("heading", { name: "Settings", level: 1 })).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "keep", confirm: "1" },
  ]);
  expect(op.sent[1]?.body).toEqual(op.sent[0]?.body);
  expect(op.sent[1]?.body).toMatchObject({ name: "1080p", cutoff: "bluray-1080p" });
});

test("an Impact that shows nothing saves without the sheet", async () => {
  const op = saving(() => noImpact);
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [op.handler, ...targetHandlers()],
  });
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "1080p web");
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));

  await expect.element(page.getByText("Saved 1080p web.")).toBeVisible();
  expect(sheet().query()).toBeNull();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "0" },
  ]);
});

test("a save refused as impact_changed previews again and shows the fresh counts, saying they changed", async () => {
  const fresh: TargetsImpact = { ...oneUnwanted, unwanted: { files: 3, bytes: 3 * 734_003_200 } };
  const op = saving((query, calls) => {
    if (query.get("preview") === "true") {
      return calls === 0 ? oneUnwanted : fresh;
    }
    return calls === 1 ? "changed" : fresh;
  });
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [op.handler, ...targetHandlers()],
  });
  await editAndSave();
  await userEvent.click(sheet().getByRole("button", { name: "Save and delete 1 file" }));

  await expect
    .element(
      sheet().getByText("These counts changed since the last preview. Check them before you save."),
    )
    .toBeVisible();
  expect(lines()).toContain("3 files would fill nothing (2.1 GB).");
  await userEvent.click(sheet().getByRole("button", { name: "Save and delete 3 files" }));

  await expect.element(page.getByText("Saved 1080p.")).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "1" },
    { preview: "true" },
    { unwanted: "delete", confirm: "3" },
  ]);
});

test("Cancel on the Impact saves nothing and keeps the edit", async () => {
  const op = saving(() => oneUnwanted);
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [op.handler, ...targetHandlers()],
  });
  await editAndSave();
  await userEvent.click(sheet().getByRole("button", { name: "Cancel" }));

  await expect.element(sheet()).not.toBeInTheDocument();
  await expect.element(page.getByRole("checkbox", { name: "WEBDL-1080p" })).not.toBeChecked();
  expect(op.sent.map((call) => call.query)).toEqual([{ preview: "true" }]);
});
