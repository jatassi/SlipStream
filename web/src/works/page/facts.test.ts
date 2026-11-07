import { expect, test } from "vitest";

import type { WorksContentType, WorksFieldSpec, WorksFieldValue } from "../../api/gen/model";
import { cinema, orchard } from "../../test/library";
import { factTexts, factsLine } from "./facts";

const genres: WorksFieldSpec = {
  key: "genres",
  label: "Genres",
  tier: "work",
  kind: "texts",
  in: ["facts"],
  list: { column: false, sort: false, filter: false },
  with: null,
  art: null,
};

const withGenres: WorksContentType = { ...cinema, fields: [...cinema.fields, genres] };

test("the facts line puts the type's noun after the first fact, in declared order", () => {
  const fields: WorksFieldValue[] = [
    ...orchard.fields,
    { key: "genres", kind: "texts", texts: ["Crime", "Drama"] },
  ];
  expect(factTexts(withGenres, fields)).toEqual(["2021", "1h 52m", "Crime, Drama"]);
  expect(factsLine(withGenres, fields)).toBe("2021 · Film · 1h 52m · Crime, Drama");
});

test("a Work with no facts gives the type's noun alone", () => {
  expect(factTexts(cinema, [])).toEqual([]);
  expect(factsLine(cinema, [])).toBe("Film");
});
