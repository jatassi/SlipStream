import { expect, test } from "vitest";

import { films, ultra } from "../../test/profiles";
import { archiveRoot, filmsRoot } from "../../test/roots";
import { movedDefaults, startChoices } from "./choices";

const defaults = { profileId: films.id, rootId: filmsRoot.id, monitor: true };

test("choices start on the defaults with Remember on, and move nothing", () => {
  const choices = startChoices(defaults);
  expect(choices).toEqual({ ...defaults, remember: true });
  expect(movedDefaults(choices, defaults)).toEqual([]);
});

test("each changed choice moves its default while Remember is on", () => {
  const changed = {
    profileId: ultra.id,
    rootId: archiveRoot.id,
    monitor: false,
    remember: true,
  };
  expect(movedDefaults(changed, defaults)).toEqual(["profile", "root", "monitor"]);
  expect(movedDefaults({ ...startChoices(defaults), monitor: false }, defaults)).toEqual([
    "monitor",
  ]);
});

test("with Remember off nothing moves", () => {
  expect(
    movedDefaults({ ...startChoices(defaults), profileId: ultra.id, remember: false }, defaults),
  ).toEqual([]);
});
