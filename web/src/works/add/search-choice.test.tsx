import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { addServer, anthologyCatalog } from "../../test/adding";
import { longShelf, longShelfCells } from "../../test/anthology";
import { renderApp } from "../../test/render-app";
import { workServer } from "../../test/work-server";

const sheetOf = (label: string) => page.getByRole("dialog", { name: `Add ${label}` });
const shelfSheet = () => sheetOf("The Long Shelf");
const searchPicker = () => shelfSheet().getByRole("combobox", { name: "Search after adding" });
const mismatch = (where: string) =>
  shelfSheet().getByText(`${where} is monitored, so SlipStream won't search after adding.`);

function openShelf(catalog = anthologyCatalog()) {
  const server = addServer({ catalogs: [catalog], added: longShelf });
  renderApp({
    path: "/add?contentType=anthology&query=shelf",
    handlers: [
      ...server.handlers,
      ...workServer({ work: longShelf, startCells: longShelfCells }).handlers,
    ],
  });
  return server;
}

async function choose(combobox: string, option: string | RegExp) {
  await userEvent.click(shelfSheet().getByRole("combobox", { name: combobox }));
  await userEvent.click(page.getByRole("option", { name: option }));
}

test("the search choice lists the type's scopes and starts on its stored default", async () => {
  openShelf(anthologyCatalog({ search: "first_item" }));
  await userEvent.click(page.getByRole("button", { name: "Add The Long Shelf" }));
  await expect.element(searchPicker()).toHaveTextContent("First story");
  await userEvent.click(searchPicker());
  await expect.element(page.getByRole("option").first()).toBeVisible();
  expect(
    page
      .getByRole("option")
      .all()
      .map((option) => option.element().textContent),
  ).toEqual(["Everything monitored", "First volume", "Latest volume", "First story", "Nothing"]);
});

test("a search that misses what the preset monitors says so under the choice", async () => {
  const server = openShelf();
  await userEvent.click(page.getByRole("button", { name: "Add The Long Shelf" }));
  await expect.element(searchPicker()).toBeVisible();
  expect(
    shelfSheet()
      .getByText(/so SlipStream won't search after adding\.$/u)
      .query(),
  ).toBeNull();

  await choose("Stories to monitor", /^First volume · /u);
  await choose("Search after adding", "Latest volume");
  await expect.element(mismatch("Nothing in the latest volume")).toBeVisible();

  await choose("Stories to monitor", /^None · /u);
  await choose("Search after adding", "Everything monitored");
  await expect.element(mismatch("Nothing")).toBeVisible();

  await choose("Search after adding", "Nothing");
  expect(
    shelfSheet()
      .getByText(/so SlipStream won't search after adding\.$/u)
      .query(),
  ).toBeNull();

  await userEvent.click(shelfSheet().getByRole("button", { name: "Add Anthology" }));
  await expect.poll(() => server.posted).toHaveLength(1);
  expect(server.posted[0]).toMatchObject({ preset: "none", search: "none" });
  await expect.element(page.getByText("Added The Long Shelf", { exact: true })).toBeVisible();
});

test("an add whose search finds Items says it is searching now", async () => {
  const server = openShelf();
  await userEvent.click(page.getByRole("button", { name: "Add The Long Shelf" }));
  await choose("Search after adding", "First volume");
  await userEvent.click(shelfSheet().getByRole("button", { name: "Add Anthology" }));
  await expect.element(page.getByText("Added The Long Shelf. Searching now.")).toBeVisible();
  expect(server.posted[0]).toMatchObject({ preset: "all", search: "first_level" });
});

test("a one-Item type asks with a switch, and warns while it isn't monitored", async () => {
  const server = addServer();
  renderApp({ path: "/add?contentType=cinema&query=quiet", handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Add The Quiet Orchard" }));
  const orchard = sheetOf("The Quiet Orchard");
  await expect.element(orchard.getByRole("switch", { name: "Search after adding" })).toBeChecked();

  await userEvent.click(orchard.getByRole("switch", { name: "Monitor" }));
  await expect
    .element(orchard.getByText("Nothing is monitored, so SlipStream won't search after adding."))
    .toBeVisible();
  await userEvent.click(orchard.getByRole("switch", { name: "Search after adding" }));
  await userEvent.click(orchard.getByRole("button", { name: "Add Film" }));
  await expect.poll(() => server.posted).toHaveLength(1);
  expect(server.posted[0]).toMatchObject({ preset: "none", search: "none" });
});
