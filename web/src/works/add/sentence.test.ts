import { expect, test } from "vitest";

import type { TargetsProfile, WorksFieldValue } from "../../api/gen/model";
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
const context = { type: cinema, profiles: [films, ultra], roots: [filmsRoot] };
const defaults = startChoices({ profileId: films.id, rootId: filmsRoot.id, monitor: true });

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
    { ...defaults, profileId: ultra.id, monitor: false },
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
