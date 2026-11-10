import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { anthologyId, longShelf, longShelfCells } from "../../test/anthology";
import { orchardId } from "../../test/library";
import { optionsServer } from "../../test/monitoring";
import { renamesServer, row } from "../../test/renames";
import { renderApp } from "../../test/render-app";
import { scannedFiles } from "../../test/work-cells";
import { workServer } from "../../test/work-server";

const edit = page.getByRole("dialog", { name: "Edit The Long Shelf" });
const confirm = page.getByRole("dialog", { name: "Change the story order to Reading order?" });
const order = edit.getByRole("radiogroup", { name: "Story order" });
const byPrinting = edit.getByRole("switch", { name: "Name by first printing" });

const plan = renamesServer({
  plan: [
    {
      workId: anthologyId,
      contentType: "anthology",
      title: "The Long Shelf",
      digest: "dddddddddddddddd",
      rows: [
        row(
          "0c8e3d2f-5e6b-4f7c-9d0e-2b3c4d5e6f21",
          "The Long Shelf/Volume 01/The Long Shelf - 1x01.mkv",
          "The Long Shelf/The Long Shelf - 1x01.mkv",
        ),
      ],
    },
  ],
});

async function openEdit(options: Parameters<typeof optionsServer>[0]) {
  const server = optionsServer(options);
  renderApp({
    path: `/works/${anthologyId}`,
    handlers: [
      server.handler,
      ...workServer({ work: longShelf, startCells: longShelfCells, files: scannedFiles }).handlers,
      ...plan.handlers,
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "More actions for The Long Shelf" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Edit…" }));
  return server;
}

test("Edit shows the type's options and the level folders of a Work of many Items", async () => {
  await openEdit({ withdraws: 0 });
  await expect.element(edit.getByRole("switch", { name: "Volume folders" })).toBeChecked();
  await expect.element(order.getByRole("radio", { name: "Published order" })).toBeChecked();
  await expect.element(byPrinting).not.toBeChecked();
});

test("a switch that moves nothing saves at once, with no confirmation", async () => {
  const server = await openEdit({ withdraws: 0 });
  await userEvent.click(byPrinting);
  await expect.element(byPrinting).toBeChecked();
  expect(page.getByRole("dialog", { name: /^Change/u }).query()).toBeNull();
  expect(server.sent).toEqual([
    { type: { byPrinting: true }, preview: true },
    { type: { byPrinting: true }, preview: false },
  ]);
});

test("a switch the type refuses shows the refusal on the field", async () => {
  const sentence =
    "Reading order names stories by number. Turn off Name by first printing to use it.";
  const server = await openEdit({ withdraws: 0, refused: { byPrinting: sentence } });
  await userEvent.click(byPrinting);
  await expect.element(edit.getByText(sentence)).toBeVisible();
  await expect.element(byPrinting).not.toBeChecked();
  expect(server.sent).toEqual([{ type: { byPrinting: true }, preview: true }]);
});

test("an order change previews what it withdraws and saves once confirmed", async () => {
  const server = await openEdit({ withdraws: 3 });
  await userEvent.click(order.getByRole("radio", { name: "Reading order" }));

  await expect
    .element(
      confirm.getByText(
        "Changing the story order moves stories between volumes. Each story keeps its files, Grabs and monitoring.",
      ),
    )
    .toBeVisible();
  await expect
    .element(
      confirm.getByText(
        "3 stories aren't in Reading order, so they'll show as No longer listed by Fixture source.",
      ),
    )
    .toBeVisible();
  await expect
    .element(confirm.getByText("File names keep the old numbering until you rename the files."))
    .toBeVisible();
  expect(server.sent).toEqual([{ type: { order: "reading" }, preview: true }]);

  await userEvent.click(confirm.getByRole("button", { name: "Change story order" }));
  await expect.element(confirm).not.toBeInTheDocument();
  await expect.element(order.getByRole("radio", { name: "Reading order" })).toBeChecked();
  expect(server.sent).toEqual([
    { type: { order: "reading" }, preview: true },
    { type: { order: "reading" }, preview: false },
  ]);
});

test("canceling the confirmation saves nothing and keeps the order", async () => {
  const server = await openEdit({ withdraws: 0 });
  await userEvent.click(order.getByRole("radio", { name: "Reading order" }));
  await expect.element(confirm.getByText(/aren't in Reading order/u)).not.toBeInTheDocument();
  await userEvent.click(confirm.getByRole("button", { name: "Cancel" }));
  await expect.element(confirm).not.toBeInTheDocument();
  await expect.element(order.getByRole("radio", { name: "Published order" })).toBeChecked();
  expect(server.sent).toEqual([{ type: { order: "reading" }, preview: true }]);
});

test("an order the source doesn't list is refused on the field", async () => {
  await openEdit({ withdraws: 0, unavailable: ["reading"] });
  await userEvent.click(order.getByRole("radio", { name: "Reading order" }));
  await expect
    .element(
      edit.getByText(
        "Fixture source doesn't list this anthology in Reading order. Choose another story order.",
      ),
    )
    .toBeVisible();
  await expect.element(confirm).not.toBeInTheDocument();
});

test("Save sends a level folders change, then links to the Work's rename plan", async () => {
  const server = await openEdit({ withdraws: 0 });
  await userEvent.click(edit.getByRole("switch", { name: "Volume folders" }));
  await userEvent.click(edit.getByRole("button", { name: "Save" }));
  await expect
    .element(edit.getByText("Files keep their names and folders until you rename them."))
    .toBeVisible();
  expect(server.sent).toEqual([{ levelFolders: false, preview: false }]);

  await userEvent.click(edit.getByRole("button", { name: "Rename files" }));
  const sheet = page.getByRole("dialog", { name: "Rename files of The Long Shelf" });
  await expect.element(sheet.getByText("This renames 1 file.")).toBeVisible();
  await expect
    .element(sheet.getByRole("insertion"))
    .toHaveTextContent("The Long Shelf/The Long Shelf - 1x01.mkv");
  await expect.element(edit).not.toBeInTheDocument();
});

test("an option saved links to the Work's rename plan", async () => {
  await openEdit({ withdraws: 0 });
  await userEvent.click(byPrinting);
  await expect
    .element(edit.getByText("Files keep their names and folders until you rename them."))
    .toBeVisible();
});

test("Rename files in the Work's menu opens the Work's rename plan", async () => {
  await openEdit({ withdraws: 0 });
  await userEvent.keyboard("{Escape}");
  await userEvent.click(page.getByRole("button", { name: "More actions for The Long Shelf" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Rename files…" }));
  const sheet = page.getByRole("dialog", { name: "Rename files of The Long Shelf" });
  await expect.element(sheet.getByText("This renames 1 file.")).toBeVisible();
  await userEvent.click(sheet.getByRole("button", { name: "Rename 1 file" }));
  await expect.element(sheet).not.toBeInTheDocument();
  expect(plan.confirmed.at(-1)).toEqual([{ work: anthologyId, digest: "dddddddddddddddd" }]);
});

test("a Work of one Item has no level folders and no options", async () => {
  renderApp({ path: `/works/${orchardId}`, handlers: workServer().handlers });
  await userEvent.click(page.getByRole("button", { name: "More actions for The Quiet Orchard" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Edit…" }));
  const dialog = page.getByRole("dialog", { name: "Edit The Quiet Orchard" });
  await expect.element(dialog.getByRole("combobox", { name: "Profile" })).toBeVisible();
  await expect.element(dialog.getByRole("switch")).not.toBeInTheDocument();
});
