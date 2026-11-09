import { expect, test } from "vitest";

import type { NamingTypeNaming, SettingField } from "../api/gen/model";
import { cinema } from "../test/library";
import { namingSettings, withValue } from "../test/naming";
import { recommend } from "./recommendation";

const settings = namingSettings("cinema", [
  ["folder", "Film folder", "{Film Title} {(Year)}"],
  ["file.standard", "Film file", "{Film Title} {(Year)} - {Edition Tag} {Target}"],
]);

const off = (fields: SettingField[]) =>
  fields.map((field) => (field.kind === "switch" ? withValue(field, false) : field));

const naming = (part: Partial<NamingTypeNaming>): NamingTypeNaming => ({
  contentType: "cinema",
  settings,
  problems: [],
  ...part,
});

const lost = {
  key: "cinema.naming.file.standard",
  sentence: "The file name has no {Target}. Add {Target} so each file of a Work gets its own name.",
};

test("a card leads with turning renaming on, then the recommended templates, then the files that don't match, then that all is well", () => {
  expect(recommend(cinema, naming({ settings: off(settings), problems: [lost] }), 128)).toEqual({
    sentence: "Turn on renaming so Jellyfin and Emby show the files of each film as one item.",
    action: {
      kind: "save",
      label: "Turn on renaming",
      values: { "cinema.naming.renameFiles": true },
    },
  });
  expect(recommend(cinema, naming({ problems: [lost] }), 128)).toEqual({
    sentence:
      "Jellyfin and Emby may show the files of a film as separate items with these templates. Use the recommended templates.",
    action: {
      kind: "save",
      label: "Use recommended",
      values: { "cinema.naming.file.standard": null },
    },
  });
  expect(recommend(cinema, naming({}), 1204)).toEqual({
    sentence: "1,204 film files don't match the templates.",
    action: { kind: "review", label: "Review renames" },
  });
  expect(recommend(cinema, naming({}), 1).sentence).toBe(
    "1 film file doesn't match the templates.",
  );
  expect(recommend(cinema, naming({}), 0)).toEqual({
    sentence: "Plex, Jellyfin and Emby show the files of each film as one item.",
  });
});

test("Use recommended puts back every template that differs from its default", () => {
  const changed = settings.map((field) =>
    field.key === "cinema.naming.folder" ? withValue(field, "{Film Title}") : field,
  );
  expect(recommend(cinema, naming({ settings: changed, problems: [lost] }), 0).action).toEqual({
    kind: "save",
    label: "Use recommended",
    values: { "cinema.naming.folder": null, "cinema.naming.file.standard": null },
  });
});
