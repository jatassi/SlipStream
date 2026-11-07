import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { catalogServer, readableAddress } from "../../test/catalog-server";
import { indexersServer } from "../../test/indexers-server";
import { renderApp } from "../../test/render-app";

const path = "/settings/indexers";
const catalog = () => page.getByRole("region", { name: "Indexer catalog" });

function render(options: { inUse?: string } = {}) {
  const server = catalogServer(options);
  renderApp({ path, handlers: [...server.handlers, ...indexersServer().handlers] });
  return server;
}

test("the catalog leads with when it refreshed, and Refresh now runs the refresh", async () => {
  const server = render();
  await expect
    .element(
      catalog().getByText(
        /^SlipStream last refreshed the Indexer catalog at .+\. It refreshes again at .+\.$/u,
      ),
    )
    .toBeVisible();
  expect(catalog().getByRole("textbox", { name: "Address" }).query()).toBeNull();
  await userEvent.click(catalog().getByRole("button", { name: "Refresh now" }));
  await expect
    .poll(() => server.taskWrites.at(-1))
    .toEqual({ method: "POST", path: "indexer-catalog-refresh/runs", body: undefined });
});

test("under Details the address saves only where a catalog is read, beside the refresh's cadence", async () => {
  const server = render();
  await userEvent.click(catalog().getByRole("button", { name: "Details" }));
  await expect
    .element(catalog().getByRole("button", { name: "Runs every: Every day" }))
    .toBeVisible();
  const address = catalog().getByRole("textbox", { name: "Address" });
  await userEvent.fill(address, "https://elsewhere.example.test");
  await userEvent.click(catalog().getByRole("button", { name: "Save" }));
  await expect
    .element(
      catalog().getByText(
        "SlipStream can't read an Indexer catalog at that address. Check the address and that it serves schema 12.",
      ),
    )
    .toBeVisible();
  await userEvent.fill(address, readableAddress);
  await userEvent.click(catalog().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved the Indexer catalog's address")).toBeVisible();
  await userEvent.click(catalog().getByRole("button", { name: "Use the default" }));
  await expect
    .poll(() => server.writes.at(-1))
    .toEqual({ method: "PUT", path: "address", body: { address: null } });
});

test("a custom definition deletes after asking, and one an Indexer runs keeps the dialog with why", async () => {
  const server = render({ inUse: "marsh" });
  const custom = catalog().getByRole("list", { name: "Custom definitions" });
  await expect.element(custom.getByText("Runs in place of the catalog's Marsh")).toBeVisible();
  await userEvent.click(custom.getByRole("button", { name: "Delete Marsh" }));
  const dialog = page.getByRole("dialog");
  await expect
    .element(dialog.getByText("Add Indexer offers the catalog's Marsh again."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Delete" }));
  await expect
    .element(dialog.getByText("An Indexer runs this definition. Delete the Indexer first."))
    .toBeVisible();
  expect(server.writes).toEqual([{ method: "DELETE", path: "custom/marsh" }]);
});

test("deleting the last custom definition leaves the way to add one", async () => {
  render();
  const custom = catalog().getByRole("list", { name: "Custom definitions" });
  await userEvent.click(custom.getByRole("button", { name: "Delete Marsh" }));
  await userEvent.click(page.getByRole("dialog").getByRole("button", { name: "Delete" }));
  await expect.element(page.getByText("Deleted the custom definition Marsh")).toBeVisible();
  await expect
    .element(catalog().getByText(/^No custom definitions yet\. A custom definition runs a site/u))
    .toBeVisible();
  await expect
    .element(catalog().getByRole("link", { name: "Add a custom definition" }))
    .toBeVisible();
});
