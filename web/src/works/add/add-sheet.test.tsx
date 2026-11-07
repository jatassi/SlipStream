import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { addServer, anthologiesRoot, anthologyCatalog, shelves } from "../../test/adding";
import { anthologyId, longShelf, longShelfCells } from "../../test/anthology";
import { films } from "../../test/profiles";
import { renderApp } from "../../test/render-app";
import { filmsRoot } from "../../test/roots";
import { workServer } from "../../test/work-server";

const sheetOf = (label: string) => page.getByRole("dialog", { name: `Add ${label}` });
const sheet = () => sheetOf("The Long Shelf");
const sentence = () => sheet().getByText(/^Adds /u);
const presetPicker = () => sheet().getByRole("combobox", { name: "Stories to monitor" });
const extras = () => sheet().getByRole("switch", { name: "Include extras" });
const folders = () => sheet().getByRole("switch", { name: "Volume folders" });
const willStart = () => sheet().getByText("New Anthologies will start with these.");

const says = (monitoring: string) =>
  `Adds The Long Shelf to Anthologies on /srv/media/anthologies, ${monitoring}, wanting Shelves' 1 Target.`;

function openShelf(catalog = anthologyCatalog()) {
  const server = addServer({ catalogs: [catalog], added: longShelf });
  const app = renderApp({
    path: "/add?contentType=anthology&query=shelf",
    handlers: [
      ...server.handlers,
      ...workServer({ work: longShelf, startCells: longShelfCells }).handlers,
    ],
  });
  return { server, app };
}

async function openSheet() {
  await userEvent.click(page.getByRole("button", { name: "Add The Long Shelf" }));
  await expect.element(sentence()).toBeVisible();
}

async function pick(preset: string) {
  await userEvent.click(presetPicker());
  await userEvent.click(page.getByRole("option", { name: new RegExp(`^${preset} · `, "u") }));
}

const optionTexts = () =>
  page
    .getByRole("option")
    .all()
    .map((option) => option.element().textContent);

test("the sheet counts the volumes and stories and what each preset monitors", async () => {
  openShelf();
  await openSheet();

  await expect.element(sheet().getByText("Harbor Press · 24 volumes · 1176 stories")).toBeVisible();
  await expect
    .element(sentence())
    .toHaveTextContent(says("monitoring all 1176 stories and new ones as they come out"));
  await expect.element(extras()).not.toBeChecked();
  await expect.element(folders()).toBeChecked();

  await userEvent.click(presetPicker());
  await expect.element(page.getByRole("option").first()).toBeVisible();
  expect(optionTexts()).toEqual([
    "All · 1176 stories",
    "Future · 0 stories",
    "Missing · 1176 stories",
    "Existing · 0 stories",
    "First volume · 49 stories",
    "Latest volume · 49 stories",
    "None · 0 stories",
  ]);
});

test("each preset says what it monitors now and whether it monitors new stories", async () => {
  openShelf();
  await openSheet();

  await pick("First volume");
  await expect.element(sentence()).toHaveTextContent(says("monitoring 49 stories and no new ones"));
  await pick("Future");
  await expect
    .element(sentence())
    .toHaveTextContent(says("monitoring only new stories as they come out"));
  await pick("None");
  await expect.element(sentence()).toHaveTextContent(says("monitoring no stories"));
});

test("Include extras adds the extras to every preset's count", async () => {
  openShelf();
  await openSheet();

  await userEvent.click(extras());
  await expect
    .element(sentence())
    .toHaveTextContent(says("monitoring all 1200 stories and new ones as they come out"));
  await pick("First volume");
  await expect.element(sentence()).toHaveTextContent(says("monitoring 73 stories and no new ones"));
  await expect.element(presetPicker()).toHaveTextContent("First volume · 73 stories");
});

test("Remember writes the preset, Include extras and Volume folders with the Profile and root", async () => {
  const { server, app } = openShelf();
  await openSheet();
  expect(willStart().query()).toBeNull();

  await pick("Latest volume");
  await userEvent.click(extras());
  await userEvent.click(folders());
  await expect.element(willStart()).toBeVisible();
  await userEvent.click(sheet().getByRole("button", { name: "Add Anthology" }));

  await expect.poll(() => app.router.state.location.pathname).toBe(`/works/${anthologyId}`);
  expect(server.posted).toEqual([
    {
      contentType: "anthology",
      key: "a1",
      profileId: shelves.id,
      rootId: anthologiesRoot.id,
      preset: "latest",
      includeAsides: true,
      levelFolders: false,
      remember: true,
    },
  ]);
});

test("the sheet starts on the type's stored add defaults and moves nothing", async () => {
  const { server } = openShelf(
    anthologyCatalog({ preset: "first", asides: true, levelFolders: false }),
  );
  await openSheet();

  await expect.element(presetPicker()).toHaveTextContent("First volume · 73 stories");
  await expect.element(extras()).toBeChecked();
  await expect.element(folders()).not.toBeChecked();
  await expect.element(sentence()).toHaveTextContent(says("monitoring 73 stories and no new ones"));
  expect(willStart().query()).toBeNull();

  await userEvent.click(sheet().getByRole("button", { name: "Add Anthology" }));
  await expect.poll(() => server.posted).toHaveLength(1);
  expect(server.posted[0]).toMatchObject({
    preset: "first",
    includeAsides: true,
    levelFolders: false,
  });
});

test("a one-Item type keeps the Monitor switch, which posts all or none and nothing else", async () => {
  const server = addServer();
  renderApp({ path: "/add?contentType=cinema&query=quiet", handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Add The Quiet Orchard" }));
  const orchard = sheetOf("The Quiet Orchard");

  await userEvent.click(orchard.getByRole("switch", { name: "Monitor" }));
  await expect
    .element(orchard.getByText(/^Adds /u))
    .toHaveTextContent(
      "Adds The Quiet Orchard (2021) to Films on /srv/media/films, not monitored, wanting Films' 1 Target.",
    );
  expect(orchard.getByRole("combobox", { name: /to monitor$/u }).query()).toBeNull();
  expect(orchard.getByRole("switch", { name: /^Include /u }).query()).toBeNull();
  expect(orchard.getByRole("switch", { name: / folders$/u }).query()).toBeNull();

  await userEvent.click(orchard.getByRole("button", { name: "Add Film" }));
  await expect.poll(() => server.posted).toHaveLength(1);
  expect(server.posted).toEqual([
    {
      contentType: "cinema",
      key: "9000001",
      profileId: films.id,
      rootId: filmsRoot.id,
      preset: "none",
      remember: true,
    },
  ]);
});
