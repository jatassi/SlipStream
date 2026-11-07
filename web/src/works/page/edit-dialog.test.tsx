import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { anthologyId, longShelf, longShelfCells } from "../../test/anthology";
import { orchardId } from "../../test/library";
import { optionsServer } from "../../test/monitoring";
import { renderApp } from "../../test/render-app";
import { workServer } from "../../test/work-server";

const edit = page.getByRole("dialog", { name: "Edit The Long Shelf" });
const confirm = page.getByRole("dialog", { name: "Change the story order to Reading order?" });
const order = edit.getByRole("radiogroup", { name: "Story order" });

async function openEdit(options: Parameters<typeof optionsServer>[0]) {
  const server = optionsServer(options);
  renderApp({
    path: `/works/${anthologyId}`,
    handlers: [
      server.handler,
      ...workServer({ work: longShelf, startCells: longShelfCells }).handlers,
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
    .element(
      confirm.getByText(
        "File names change to the new numbering when SlipStream renames files. Renaming comes in a later release.",
      ),
    )
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

test("cancelling the confirmation saves nothing and keeps the order", async () => {
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

test("Save sends a level folders change", async () => {
  const server = await openEdit({ withdraws: 0 });
  await userEvent.click(edit.getByRole("switch", { name: "Volume folders" }));
  await userEvent.click(edit.getByRole("button", { name: "Save" }));
  await expect.element(edit).not.toBeInTheDocument();
  expect(server.sent).toEqual([{ levelFolders: false, preview: false }]);
});

test("a Work of one Item has no level folders and no options", async () => {
  renderApp({ path: `/works/${orchardId}`, handlers: workServer().handlers });
  await userEvent.click(page.getByRole("button", { name: "More actions for The Quiet Orchard" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Edit…" }));
  const dialog = page.getByRole("dialog", { name: "Edit The Quiet Orchard" });
  await expect.element(dialog.getByRole("combobox", { name: "Profile" })).toBeVisible();
  await expect.element(dialog.getByRole("switch")).not.toBeInTheDocument();
});
