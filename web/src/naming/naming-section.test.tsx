import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { NamingProblem, SettingField } from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { anthologies } from "../test/anthology";
import { cinema } from "../test/library";
import {
  type NamingServerType,
  fileTokens,
  filmTokens,
  namingServer,
  namingSettings,
  withValue,
} from "../test/naming";
import { renderApp } from "../test/render-app";

const fileKey = "cinema.naming.file.standard";
const defaultFile = "{Film Title} {(Year)} - {Edition Tag} {Target}";
const noTarget =
  "The file name has no {Target}. Add {Target} so each file of a Work gets its own name.";

const renamingOff = (settings: SettingField[]) =>
  settings.map((field) => (field.kind === "switch" ? withValue(field, false) : field));

function films(rename = true): NamingServerType {
  const settings = namingSettings("cinema", [
    ["folder", "Film folder", "{Film Title} {(Year)}"],
    ["file.standard", "Film file", defaultFile],
  ]);
  return {
    naming: {
      contentType: "cinema",
      settings: rename ? settings : renamingOff(settings),
      problems: [],
    },
    tokens: [
      { key: "cinema.naming.folder", tokens: filmTokens },
      { key: fileKey, tokens: fileTokens },
    ],
    check: (values): NamingProblem[] =>
      values["cinema.naming.renameFiles"] === true && !String(values[fileKey]).includes("{Target}")
        ? [{ key: fileKey, sentence: noTarget }]
        : [],
  };
}

const stories: NamingServerType = {
  naming: {
    contentType: "anthology",
    settings: namingSettings(
      "anthology",
      [
        ["folder", "Anthology folder", "{Anthology Title}"],
        ["file.standard", "Story file", "{Anthology Title} - {volume}x{story:00} - {Target}"],
      ],
      true,
    ),
    problems: [],
  },
  tokens: [],
};

const shelves: NamingServerType = {
  naming: { ...stories.naming, contentType: "shelf" },
  tokens: [],
};

function openNaming(types: readonly NamingServerType[]) {
  const server = namingServer(types);
  renderApp({
    path: "/settings/naming",
    handlers: [
      getListContentTypesMockHandler({
        contentTypes: [
          cinema,
          anthologies,
          { ...anthologies, id: "shelf", name: "Shelves", enabled: false },
        ],
      }),
      ...server.handlers,
    ],
  });
  return server;
}

async function openDetails(name: string) {
  const card = page.getByRole("article", { name });
  await userEvent.click(card.getByRole("button", { name: "Details" }));
  return card;
}

test("each enabled Content Type's card leads with what to do, and its button saves it", async () => {
  const { sent } = openNaming([films(false), stories, shelves]);
  const filmCard = page.getByRole("article", { name: "Films" });
  await expect
    .element(
      filmCard.getByText(
        "Turn on renaming so Jellyfin and Emby show the files of each film as one item.",
      ),
    )
    .toBeVisible();
  await expect
    .element(
      page
        .getByRole("article", { name: "Anthologies" })
        .getByText("Plex, Jellyfin and Emby show the files of each anthology as one item."),
    )
    .toBeVisible();
  expect(page.getByRole("article", { name: "Shelves" }).query()).toBeNull();

  await userEvent.click(filmCard.getByRole("button", { name: "Turn on renaming" }));
  await expect
    .element(filmCard.getByText("Plex, Jellyfin and Emby show the files of each film as one item."))
    .toBeVisible();
  expect(sent).toEqual([{ "cinema.naming.renameFiles": true }]);
  expect(filmCard.getByRole("button", { name: "Turn on renaming" }).query()).toBeNull();
});

test("Details holds the rename switch, the templates, the colon mode and, for numbered Items, the multi-episode style", async () => {
  openNaming([films(false), stories]);
  const filmCard = await openDetails("Films");
  await expect.element(filmCard.getByRole("switch", { name: "Rename files" })).not.toBeChecked();
  await expect
    .element(
      filmCard.getByText(
        "With renaming off, Jellyfin and Emby may show the files of a film as separate items. Imports keep the name of the downloaded file.",
      ),
    )
    .toBeVisible();
  await expect
    .element(filmCard.getByRole("combobox", { name: "Film folder" }))
    .toHaveValue("{Film Title} {(Year)}");
  await expect.element(filmCard.getByRole("combobox", { name: "Colon replacement" })).toBeVisible();
  expect(filmCard.getByRole("combobox", { name: "Multi-story style" }).query()).toBeNull();

  const storyCard = await openDetails("Anthologies");
  await expect
    .element(storyCard.getByRole("combobox", { name: "Multi-story style" }))
    .toBeVisible();
});

test("a template previews the draft's paths and the save check's sentence, and a refused save says why", async () => {
  const { sent } = openNaming([films()]);
  const card = await openDetails("Films");
  const file = card.getByRole("combobox", { name: "Film file" });
  const samples = card.getByRole("list", { name: "Film file samples" });
  await expect.element(samples.getByText(`${defaultFile}.mkv`)).toBeVisible();

  await userEvent.fill(file, "{Film Title} {(Year)}");
  await expect.element(samples.getByText("{Film Title} {(Year)}.mkv")).toBeVisible();
  await expect.element(card.getByRole("alert").filter({ hasText: noTarget }).first()).toBeVisible();
  await expect.element(file).toHaveAttribute("aria-invalid", "true");

  await userEvent.click(card.getByRole("button", { name: "Save", exact: true }));
  await expect.element(card.getByText(`Film file: ${noTarget}`)).toBeVisible();
  expect(sent).toEqual([{ [fileKey]: "{Film Title} {(Year)}" }]);

  await userEvent.click(card.getByRole("button", { name: "Use recommended" }));
  await expect.element(file).toHaveValue(defaultFile);
  expect(card.getByRole("button", { name: "Save", exact: true }).query()).toBeNull();
});

test("typing { suggests the field's tokens, and Enter puts the active one in place", async () => {
  openNaming([films()]);
  const card = await openDetails("Films");
  const file = card.getByRole("combobox", { name: "Film file" });
  await userEvent.fill(file, "{Film Title} - {rel");
  const listbox = page.getByRole("listbox", { name: "Tokens" });
  const options = listbox.getByRole("option");
  await expect.element(options.first().getByText("Year", { exact: true })).toBeVisible();
  await expect.element(options.nth(1).getByText("Release Group", { exact: true })).toBeVisible();

  await userEvent.keyboard("{ArrowDown}{Enter}");
  await expect.element(file).toHaveValue("{Film Title} - {Release Group}");
  expect(page.getByRole("listbox", { name: "Tokens" }).query()).toBeNull();
});

test("the token builder builds a token from its controls and inserts it at the caret", async () => {
  openNaming([films()]);
  const card = await openDetails("Films");
  const folder = card.getByRole("combobox", { name: "Film folder" });
  await userEvent.fill(folder, "{Film Title} ");
  await userEvent.click(card.getByRole("button", { name: "Tokens for Film folder" }));

  const builder = page.getByRole("dialog", { name: "Film folder tokens" });
  await userEvent.click(builder.getByRole("button", { name: "Year" }));
  await expect
    .element(builder.getByText('The year the film came out. For example, "2021".'))
    .toBeVisible();
  await userEvent.click(builder.getByRole("combobox", { name: "Text around it" }));
  await userEvent.click(page.getByRole("option", { name: "Parentheses" }));
  await expect
    .element(builder.getByRole("status", { name: "Token text" }))
    .toHaveTextContent("{(Year)}");

  await userEvent.click(builder.getByRole("button", { name: "Insert" }));
  await expect.element(folder).toHaveValue("{Film Title} {(Year)}");
});
