import { expect, test } from "vitest";

import type { WorksContentType } from "../api/gen/model";
import { cinema, orchardRow } from "../test/library";
import { parseRow } from "./rows";

const rated: WorksContentType = {
  ...cinema,
  fields: [
    ...cinema.fields,
    {
      key: "genres",
      label: "Genres",
      tier: "work",
      kind: "texts",
      in: ["facts"],
      list: { column: false, sort: false, filter: true },
      with: null,
      art: null,
    },
  ],
};

test("a row's values are typed by its type's listed fields, and an unlisted one is dropped", () => {
  const work = parseRow(
    {
      ...orchardRow,
      values: { year: 2021, runtime: 6_720_000, released: "2021-04-09", genres: ["Drama"] },
    },
    rated,
  );
  expect(Object.fromEntries(work.values)).toEqual({
    year: { kind: "number", number: 2021 },
    runtime: { kind: "duration", ms: 6_720_000 },
    genres: { kind: "texts", texts: ["Drama"] },
  });
});

test("a value whose shape doesn't match its field's kind is dropped", () => {
  const work = parseRow({ ...orchardRow, values: { year: "2021", runtime: 6_720_000 } }, cinema);
  expect([...work.values.keys()]).toEqual(["runtime"]);
});

test("a row parses once per wire row and type, so a splice parses only what it replaced", () => {
  const first = parseRow(orchardRow, cinema);
  expect(parseRow(orchardRow, cinema)).toBe(first);
  expect(parseRow({ ...orchardRow }, cinema)).not.toBe(first);
  expect(parseRow(orchardRow, rated)).not.toBe(first);
});

test("nullable wire fields read as absent", () => {
  const work = parseRow(
    { ...orchardRow, availableOn: null, posterUrl: null, rootId: null },
    cinema,
  );
  expect([work.availableOn, work.poster, work.rootId, work.gone]).toEqual([
    undefined,
    undefined,
    undefined,
    false,
  ]);
  expect(parseRow(orchardRow, cinema).availableOn).toBe("2021-04-09");
});
