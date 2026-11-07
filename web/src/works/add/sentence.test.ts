import { expect, test } from "vitest";

import type { TargetsProfile, WorksFieldValue } from "../../api/gen/model";
import { anthologiesRoot, longShelfPresets, shelves } from "../../test/adding";
import { anthologies } from "../../test/anthology";
import { cinema } from "../../test/library";
import { films, ultra } from "../../test/profiles";
import { filmsRoot } from "../../test/roots";
import { startChoices } from "./choices";
import { addSentence } from "./sentence";

const fields: WorksFieldValue[] = [
  { key: "year", kind: "number", number: 2021 },
  { key: "runtime", kind: "duration", ms: 6_240_000 },
];
const orchard = { label: "The Quiet Orchard", fields };
const context = {
  type: cinema,
  profiles: [films, ultra],
  roots: [filmsRoot],
  monitor: { kind: "switch" },
  counts: null,
} as const;
const defaults = startChoices({
  profileId: films.id,
  rootId: filmsRoot.id,
  preset: "all",
  asides: undefined,
  levelFolders: undefined,
});

test.each([
  [
    "the defaults",
    orchard,
    defaults,
    "Adds The Quiet Orchard (2021) to Films on /srv/media/films, monitored, wanting Films' 1 Target.",
  ],
  [
    "another Profile, not monitored",
    orchard,
    { ...defaults, profileId: ultra.id, preset: "none" as const },
    "Adds The Quiet Orchard (2021) to Films on /srv/media/films, not monitored, wanting Ultra's 2 Targets.",
  ],
  [
    "no root",
    orchard,
    { ...defaults, rootId: undefined },
    "Adds The Quiet Orchard (2021) to Films with no library root, monitored, wanting Films' 1 Target.",
  ],
  [
    "no year",
    { label: "The Quiet Orchard", fields: [] },
    defaults,
    "Adds The Quiet Orchard to Films on /srv/media/films, monitored, wanting Films' 1 Target.",
  ],
])("the sentence for %s", (_case, work, choices, sentence) => {
  expect(addSentence(work, choices, context)).toBe(sentence);
});

test("a Profile with no Targets says the Work wants nothing", () => {
  const empty: TargetsProfile = {
    ...films,
    id: "0199e1f2-7a3b-7c4d-8e5f-00000000000e",
    name: "Empty",
    targets: [],
  };
  expect(
    addSentence(orchard, { ...defaults, profileId: empty.id }, { ...context, profiles: [empty] }),
  ).toBe(
    "Adds The Quiet Orchard (2021) to Films on /srv/media/films, monitored, wanting nothing, because Empty has no Targets.",
  );
});

const shelf = {
  label: "The Long Shelf",
  fields: [{ key: "publisher", kind: "text", text: "Harbor Press" }],
} satisfies { label: string; fields: WorksFieldValue[] };
const anthology = {
  type: anthologies,
  profiles: [shelves],
  roots: [anthologiesRoot],
  monitor: { kind: "choice", label: "Stories to monitor", options: [] },
  counts: longShelfPresets,
} as const;
const shelfChoices = startChoices({
  profileId: shelves.id,
  rootId: anthologiesRoot.id,
  preset: "all",
  asides: false,
  levelFolders: true,
});

test.each([
  ["all", false, "monitoring all 1176 stories and new ones as they air"],
  ["all", true, "monitoring all 1200 stories and new ones as they air"],
  ["first", false, "monitoring 49 stories and no new ones"],
  ["first", true, "monitoring 73 stories and no new ones"],
  ["future", false, "monitoring only new stories as they air"],
  ["none", false, "monitoring no stories"],
] as const)("the %s preset, extras included %s, says %s", (preset, asides, monitoring) => {
  expect(addSentence(shelf, { ...shelfChoices, preset, asides }, anthology)).toBe(
    `Adds The Long Shelf to Anthologies on /srv/media/anthologies, ${monitoring}, wanting Shelves' 1 Target.`,
  );
});

test("a Work already in the library has no counts, so the sentence leaves monitoring out", () => {
  expect(addSentence(shelf, shelfChoices, { ...anthology, counts: null })).toBe(
    "Adds The Long Shelf to Anthologies on /srv/media/anthologies, wanting Shelves' 1 Target.",
  );
});
