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

test("a card leads with turning renaming on, then the recommended templates, then that all is well", () => {
  expect(recommend(cinema, naming({ settings: off(settings), problems: [lost] }))).toEqual({
    sentence: "Turn on renaming so Jellyfin and Emby show the files of each film as one item.",
    action: { label: "Turn on renaming", values: { "cinema.naming.renameFiles": true } },
  });
  expect(recommend(cinema, naming({ problems: [lost] }))).toEqual({
    sentence:
      "Jellyfin and Emby may show the files of a film as separate items with these templates. Use the recommended templates.",
    action: { label: "Use recommended", values: { "cinema.naming.file.standard": null } },
  });
  expect(recommend(cinema, naming({}))).toEqual({
    sentence: "Plex, Jellyfin and Emby show the files of each film as one item.",
  });
});

test("Use recommended puts back every template that differs from its default", () => {
  const changed = settings.map((field) =>
    field.key === "cinema.naming.folder" ? withValue(field, "{Film Title}") : field,
  );
  expect(recommend(cinema, naming({ settings: changed, problems: [lost] })).action?.values).toEqual(
    {
      "cinema.naming.folder": null,
      "cinema.naming.file.standard": null,
    },
  );
});
