import type {
  ImportsLead,
  ImportsRootLabel,
  WorksCandidate,
  WorksContentType,
} from "../api/gen/model";
import type { RootEntry, RootFile } from "../imports/root-entry";
import { cinema, orchard } from "./library";

/** A many-Item type with seasons and an episode code, as a series type reads. */
export const shows: WorksContentType = {
  ...cinema,
  id: "show",
  name: "Shows",
  icon: "tv",
  work: { one: "Show", many: "Shows" },
  item: { one: "Episode", many: "Episodes" },
  levels: [{ one: "Season", many: "Seasons" }],
  items: "many",
  source: "TVDB",
  fields: [
    {
      key: "code",
      label: "Episode",
      tier: "item",
      kind: "text",
      in: ["code"],
      list: { column: false, sort: false, filter: false },
      with: null,
      art: null,
    },
  ],
};

export const filmsShelf: ImportsRootLabel = {
  id: "0199f0a1-1111-7000-8000-000000000001",
  label: "Films library root on home",
};
export const filmsVault: ImportsRootLabel = {
  id: "0199f0a1-1111-7000-8000-000000000002",
  label: "Films library root on vault",
};
export const showsShelf: ImportsRootLabel = {
  id: "0199f0a1-1111-7000-8000-000000000003",
  label: "Shows library root on home",
};

const gib = 1024 ** 3;

type FixtureFile = Omit<RootFile, "position" | "outcome">;

/** A root Import whose files wait at positions in the order given. */
const entryOf = ({
  files,
  ...entry
}: Pick<RootEntry, "id" | "root" | "folder" | "hint"> &
  Partial<Omit<RootEntry, "files">> & { files: FixtureFile[] }): RootEntry => ({
  source: "root",
  stage: "waiting",
  contentType: cinema.id,
  suggestion: null,
  createdAt: "2026-11-07T03:12:00Z",
  ignoredAt: null,
  finishedAt: null,
  ...entry,
  files: files.map((file, position) => ({ ...file, position, outcome: "waiting" })),
});

export const halloween = entryOf({
  id: "0199f0b2-0000-7000-8000-000000000001",
  root: filmsShelf,
  folder: "Halloween",
  hint: { title: "Halloween", year: null, ids: [] },
  suggestion: {
    key: "948",
    label: "Halloween",
    year: 1978,
    posterUrl: null,
    workId: null,
    confident: false,
  },
  files: [
    {
      path: "Halloween/Halloween.mkv",
      size: 8 * gib,
      quality: "bluray-1080p",
      cut: "standard",
      itemIds: [],
      reason: { code: "several", title: "Halloween", count: 3 },
    },
  ],
});

export const brazil = entryOf({
  id: "0199f0b2-0000-7000-8000-000000000002",
  root: filmsShelf,
  folder: "Brazil (1985)",
  hint: { title: "Brazil", year: 1985, ids: [] },
  files: [
    {
      path: "Brazil (1985)/Brazil.1985.mkv",
      size: 4 * gib,
      quality: "webdl-1080p",
      cut: "standard",
      itemIds: [],
      reason: { code: "no_match", title: "Brazil", year: 1985 },
    },
    {
      path: "Brazil (1985)/Brazil.1985.sample.mkv",
      size: 0.1 * gib,
      quality: null,
      cut: "standard",
      itemIds: [],
      reason: { code: "unreadable", line: "Invalid data found when processing input" },
    },
  ],
});

export const heat = entryOf({
  id: "0199f0b2-0000-7000-8000-000000000003",
  root: filmsShelf,
  folder: "Heat (1995)",
  hint: { title: "Heat", year: 1995, ids: [{ provider: "tmdb", value: "949" }] },
  suggestion: {
    key: "949",
    label: "Heat",
    year: 1995,
    posterUrl: null,
    workId: "0199f0c3-0000-7000-8000-000000000949",
    confident: true,
  },
  files: [
    {
      path: "Heat (1995)/Heat.mkv",
      size: 12 * gib,
      quality: "remux-1080p",
      cut: "standard",
      itemIds: [],
      reason: { code: "elsewhere", title: "Heat", year: 1995, root: filmsVault },
    },
  ],
});

export const severanceId = "0199f0c3-0000-7000-8000-000000002022";

export const severanceEntry = entryOf({
  id: "0199f0b2-0000-7000-8000-000000000004",
  root: showsShelf,
  contentType: shows.id,
  folder: "Severance (2022)",
  hint: { title: "Severance", year: 2022, ids: [] },
  suggestion: {
    key: "371980",
    label: "Severance",
    year: 2022,
    posterUrl: null,
    workId: severanceId,
    confident: true,
  },
  files: [
    {
      path: "Severance (2022)/Season 1/Severance.mkv",
      size: 2 * gib,
      quality: "webdl-2160p",
      cut: "standard",
      itemIds: [],
      reason: { code: "no_number" },
    },
    {
      path: "Severance (2022)/Season 3/Severance.S03E14.mkv",
      size: 2 * gib,
      quality: "webdl-2160p",
      cut: "standard",
      itemIds: [],
      reason: { code: "no_such_item", names: "S03E14" },
    },
  ],
});

export const orchardLeft = entryOf({
  id: "0199f0b2-0000-7000-8000-000000000005",
  root: filmsShelf,
  folder: "The Quiet Orchard (2021)",
  hint: { title: "The Quiet Orchard", year: 2021, ids: [] },
  suggestion: {
    key: orchard.key,
    label: orchard.label,
    year: 2021,
    posterUrl: null,
    workId: null,
    confident: true,
  },
  ignoredAt: "2026-11-06T20:00:00Z",
  files: [
    {
      path: "The Quiet Orchard (2021)/The Quiet Orchard.mkv",
      size: 6 * gib,
      quality: "bluray-1080p",
      cut: "standard",
      itemIds: [],
      reason: { code: "left" },
    },
  ],
});

export const filmsLead: ImportsLead = {
  root: filmsShelf,
  contentType: cinema.id,
  added: 398,
  needsYou: 3,
};

export const showsLead: ImportsLead = {
  root: showsShelf,
  contentType: shows.id,
  added: 0,
  needsYou: 1,
};

export const halloween1978: WorksCandidate = {
  key: "948",
  label: "Halloween",
  original: "",
  overview: "A killer stalks Haddonfield on Halloween night.",
  fields: [{ key: "year", kind: "number", number: 1978 }],
  posterUrl: null,
  workId: null,
  unlisted: false,
};

export const halloween2018: WorksCandidate = {
  ...halloween1978,
  key: "424139",
  overview: "Laurie Strode faces Michael Myers forty years on.",
  fields: [{ key: "year", kind: "number", number: 2018 }],
};
