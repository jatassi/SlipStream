import type { WorksContentType, WorksFieldSpec, WorksLibraryRow } from "../api/gen/model";
import { scopeOf } from "../library/arrange";
import { type LibraryWork, parseRow } from "../library/rows";
import { type ArrangeContext, type LibraryView, defaultView } from "../library/view";
import { cinema, libraryRow } from "./library";

export const listed = (
  key: string,
  label: string,
  kind: WorksFieldSpec["kind"],
  list: Partial<WorksFieldSpec["list"]>,
): WorksFieldSpec => ({
  key,
  label,
  tier: "work",
  kind,
  in: ["facts"],
  list: { column: false, sort: false, filter: false, ...list },
  with: null,
  art: null,
});

export const films: WorksContentType = {
  ...cinema,
  ratings: ["imdb"],
  fields: [
    ...cinema.fields,
    listed("certification", "Certification", "text", { column: true, sort: true, filter: true }),
    listed("genres", "Genres", "texts", { filter: true }),
  ],
};

const gib = 1024 ** 3;
const minute = 60_000;

const rows: WorksLibraryRow[] = [
  libraryRow({
    id: "a",
    key: "1",
    label: "Amélie",
    addedAt: "2026-10-20T09:00:00Z",
    availableOn: "2026-11-14",
    bytes: 60 * gib,
    files: 1,
    status: "met",
    profileId: "archive",
    rootId: "films",
    monitored: "all",
    rating: { source: "imdb", value: 8.3, scale: 10 },
    values: { year: 2001, runtime: 98 * minute, certification: "R", genres: ["Comedy", "Romance"] },
  }),
  libraryRow({
    id: "b",
    key: "2",
    label: "Heat",
    addedAt: "2026-09-03T09:00:00Z",
    availableOn: "2026-10-30",
    bytes: 12 * gib,
    files: 2,
    status: "upgradable",
    profileId: "films",
    rootId: null,
    monitored: "some",
    rating: { source: "imdb", value: 70, scale: 100 },
    values: { year: 1995, runtime: 170 * minute, certification: "R", genres: ["Crime", "Drama"] },
  }),
  libraryRow({
    id: "c",
    key: "3",
    label: "Zodiac",
    addedAt: "2025-03-01T09:00:00Z",
    availableOn: null,
    bytes: 0,
    files: 0,
    status: "failed",
    profileId: "archive",
    rootId: "films",
    monitored: "none",
    rating: null,
    values: { year: 2007, runtime: 157 * minute, certification: "PG-13", genres: ["Crime"] },
  }),
  libraryRow({
    id: "d",
    key: "4",
    label: "1917",
    addedAt: "2026-02-01T09:00:00Z",
    availableOn: "2027",
    bytes: 2 * gib,
    files: 1,
    status: "missing",
    profileId: "films",
    rootId: "films",
    monitored: "all",
    rating: { source: "imdb", value: 5.5, scale: 10 },
    values: { year: 2019 },
  }),
  libraryRow({
    id: "e",
    key: "5",
    label: "Dune",
    addedAt: "2026-10-26T08:00:00Z",
    availableOn: "2026-10-02",
    bytes: 0,
    files: 0,
    status: "unreleased",
    profileId: "films",
    rootId: null,
    monitored: "all",
    rating: null,
    values: { year: 2021, runtime: 155 * minute },
  }),
];

export const works: LibraryWork[] = rows.map((row) => parseRow(row, films));
export const types = [films];
export const scope = scopeOf(types);

export const ctx: ArrangeContext = {
  today: new Date(2026, 9, 26, 12),
  profiles: new Map([
    ["archive", "Archive"],
    ["films", "Films"],
  ]),
  roots: new Map([["films", "/media/films"]]),
  matched: undefined,
};

export const view = (change: Partial<LibraryView>): LibraryView => ({
  ...defaultView(types),
  ...change,
});
