import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type {
  FormatsParsed,
  FormatsReadFacts,
  FormatsWorkRef,
  NamingProblem,
  NamingTestName,
} from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { ParseReleasesBody } from "../api/gen/zod/formats/formats.zod";
import { anthologies } from "../test/anthology";
import { cinema, libraryRow, libraryServer } from "../test/library";
import {
  type NamingServerType,
  fileTokens,
  filmTokens,
  namingServer,
  namingSettings,
} from "../test/naming";
import { renderApp } from "../test/render-app";

const fileKey = "cinema.naming.file.standard";
const heat = { id: "0b6f2a9e-3c1d-4e5f-8a7b-9c0d1e2f3a4b", label: "Heat (1995)" };
const tower = { id: "5d4c3b2a-1f0e-4d9c-8b7a-6f5e4d3c2b1a", label: "The Tower" };
const heatName = "Heat.1995.1080p.BluRay.x264-AMIABLE.mkv";
const towerName = "The.Tower.1x14.720p.WEB-DL.mkv";
const unknownName = "Unknown.Film.2001.mkv";
const wordsOf: Readonly<Record<string, string>> = {
  "{Film Title}": "Heat",
  "{(Year)}": "(1995)",
  "{Edition Tag}": "",
  "{Target}": "1080p",
};

const films: NamingServerType = {
  naming: {
    contentType: "cinema",
    settings: namingSettings("cinema", [
      ["folder", "Film folder", "{Film Title} {(Year)}"],
      ["file.standard", "Film file", "{Film Title} {(Year)} - {Edition Tag} {Target}"],
    ]),
    problems: [],
  },
  tokens: [
    { key: "cinema.naming.folder", tokens: filmTokens },
    { key: fileKey, tokens: fileTokens },
  ],
  check: (values): NamingProblem[] =>
    String(values[fileKey]).includes("{Target}")
      ? []
      : [{ key: fileKey, sentence: "The file name has no {Target}." }],
  tested: ({ name }, values) => ({
    name,
    path: `Heat (1995)/${String(values[fileKey]).replaceAll(/\{[^}]*\}/gu, (token) => wordsOf[token] ?? token)}.mkv`,
  }),
};

const stories: NamingServerType = {
  naming: {
    contentType: "anthology",
    settings: namingSettings("anthology", [["file.standard", "Story file", "{Anthology Title}"]]),
    problems: [],
  },
  tokens: [],
  tested: ({ name }) => ({ name, path: "", unplaced: { code: "no_such_item", names: "1x14" } }),
};

const facts: FormatsReadFacts = {
  quality: "Bluray-1080p",
  source: "Bluray",
  resolution: "1080p",
  modifier: null,
  revision: null,
  group: "AMIABLE",
  cut: "standard",
  edition: null,
  hdr: [],
  video: "x264",
  audio: [],
  channels: [],
  languages: [],
  ids: {},
  titles: ["Heat"],
  year: 1995,
  marks: [],
  span: "",
};

const parsed = (name: string, works: readonly FormatsWorkRef[]): FormatsParsed => ({
  name,
  sentence: "",
  words: "",
  facts,
  work: works.length === 1 ? (works[0] ?? null) : null,
  works: [...works],
  covers: [],
  formats: [],
  targets: [],
});

const matches: Readonly<Record<string, readonly FormatsWorkRef[]>> = {
  [heatName]: [heat],
  [towerName]: [tower],
  [unknownName]: [],
  "Heat.mkv": [heat, { id: "c0ffee00-0000-4000-8000-000000000001", label: "Heat (1986)" }],
};

function openTester() {
  const server = namingServer([films, stories]);
  const library = libraryServer([
    libraryRow({ ...heat, key: "tmdb:949" }),
    libraryRow({ ...tower, key: "anthology:1", contentType: "anthology" }),
  ]);
  const parses: unknown[] = [];
  renderApp({
    path: "/settings/naming",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema, anthologies] }),
      ...server.handlers,
      library.handler,
      http.post("*/api/v1/releases/parse", async ({ request }) => {
        const body = ParseReleasesBody.parse(await request.json());
        parses.push(body);
        return HttpResponse.json({
          results: (body.names ?? []).map((name) => parsed(name, matches[name] ?? [])),
        });
      }),
    ],
  });
  return { tested: server.tested, parses };
}

async function editFilmFile(text: string) {
  const card = page.getByRole("article", { name: "Films" });
  await userEvent.click(card.getByRole("button", { name: "Details" }));
  await userEvent.fill(card.getByRole("combobox", { name: "Film file" }), text);
}

async function testNamesIn(names: readonly string[]) {
  await userEvent.click(page.getByRole("button", { name: "Test file names" }));
  const sheet = page.getByRole("dialog", { name: "Test file names" });
  await userEvent.fill(sheet.getByRole("textbox", { name: "File paths" }), names.join("\n"));
  await userEvent.click(sheet.getByRole("button", { name: "Test", exact: true }));
  return sheet.getByRole("list", { name: "Results" });
}

test("each name shows the Work it matches and the path its card's unsaved templates give it, or why it goes nowhere", async () => {
  const { tested, parses } = openTester();
  await editFilmFile("{Film Title} {(Year)} - {Target}");
  const results = await testNamesIn([heatName, towerName, unknownName]);

  await expect
    .element(results.getByText("It matches Heat (1995) and would be imported as:"))
    .toBeVisible();
  await expect.element(results.getByText("Heat (1995)/Heat (1995) - 1080p.mkv")).toBeVisible();
  await expect
    .element(results.getByText("It matches The Tower. This anthology has no story 1x14."))
    .toBeVisible();
  await expect
    .element(
      results.getByText("It matches no Work. A library scan would send it to Pending imports."),
    )
    .toBeVisible();
  expect(parses).toEqual([{ names: [heatName, towerName, unknownName], kind: "file" }]);
  expect(tested).toEqual(
    expect.arrayContaining([
      {
        type: "cinema",
        body: {
          draft: { [fileKey]: "{Film Title} {(Year)} - {Target}" },
          names: [{ name: heatName, workId: heat.id } satisfies NamingTestName],
        },
      },
      {
        type: "anthology",
        body: { draft: {}, names: [{ name: towerName, workId: tower.id }] },
      },
    ]),
  );
  expect(tested).toHaveLength(2);

  await userEvent.click(results.getByRole("button", { name: "Details" }).first());
  await expect.element(results.getByText("What it reads as")).toBeVisible();
  await expect.element(results.getByText("Bluray-1080p")).toBeVisible();
});

test("a name that matches several Works, or a card the save check refuses, says so", async () => {
  openTester();
  await editFilmFile("{Film Title} {(Year)}");
  const results = await testNamesIn([heatName, "Heat.mkv"]);

  await expect
    .element(
      results.getByText(
        "It matches Heat (1995). The Films card has a setting the save check refuses. Fix it to see where this file would go.",
      ),
    )
    .toBeVisible();
  await expect
    .element(
      results.getByText(
        "It matches 2 Works: Heat (1995) and Heat (1986). A library scan would send it to Pending imports.",
      ),
    )
    .toBeVisible();
});
