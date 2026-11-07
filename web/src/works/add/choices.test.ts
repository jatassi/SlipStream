import { expect, test } from "vitest";

import { anthologies } from "../../test/anthology";
import { addMonitor, anthologyCatalog, anthologiesRoot, shelves } from "../../test/adding";
import { cinema } from "../../test/library";
import { films, ultra } from "../../test/profiles";
import { archiveRoot, filmsRoot } from "../../test/roots";
import { type AddDefaults, addSetupOf, movedDefaults, startChoices } from "./choices";

const defaults: AddDefaults = {
  profileId: films.id,
  rootId: filmsRoot.id,
  preset: "all",
  asides: undefined,
  levelFolders: undefined,
};

const cinemaSetup = (on: boolean) =>
  addSetupOf({
    type: cinema,
    profiles: [films, ultra],
    roots: [filmsRoot],
    settings: [addMonitor(on)],
  });

test("a switch maps to all or none and asks with a switch", () => {
  expect(cinemaSetup(true)).toEqual({ defaults, monitor: { kind: "switch" } });
  expect(cinemaSetup(false).defaults.preset).toBe("none");
});

test("a choice asks with its presets and reads the aside and folder defaults", () => {
  const { settings } = anthologyCatalog({ preset: "latest", asides: true, levelFolders: false });
  const setup = addSetupOf({
    type: anthologies,
    profiles: [shelves],
    roots: [anthologiesRoot],
    settings,
  });
  expect(setup.defaults).toEqual({
    profileId: shelves.id,
    rootId: anthologiesRoot.id,
    preset: "latest",
    asides: true,
    levelFolders: false,
  });
  expect(setup.monitor).toMatchObject({ kind: "choice", label: "Stories to monitor" });
  expect(
    setup.monitor.kind === "choice" && setup.monitor.options.map(({ label }) => label),
  ).toEqual(["All", "Future", "Missing", "Existing", "First volume", "Latest volume", "None"]);
});

test("choices start on the defaults with Remember on, and move nothing", () => {
  const choices = startChoices(defaults);
  expect(choices).toEqual({ ...defaults, remember: true });
  expect(movedDefaults(choices, defaults)).toEqual([]);
});

test("each changed choice moves its default while Remember is on", () => {
  const start = { ...defaults, asides: false, levelFolders: true };
  const changed = {
    profileId: ultra.id,
    rootId: archiveRoot.id,
    preset: "first" as const,
    asides: true,
    levelFolders: false,
    remember: true,
  };
  expect(movedDefaults(changed, start)).toEqual([
    "profile",
    "root",
    "monitor",
    "asides",
    "levelFolders",
  ]);
  expect(movedDefaults({ ...startChoices(start), preset: "none" }, start)).toEqual(["monitor"]);
});

test("with Remember off nothing moves", () => {
  expect(
    movedDefaults({ ...startChoices(defaults), profileId: ultra.id, remember: false }, defaults),
  ).toEqual([]);
});
