import { describe, expect, test } from "vitest";

import { suggestPlace } from "./language";

const languages = ["de-DE", "en-GB", "en-US", "fr-CA", "fr-FR", "pt-BR", "zh-TW"];
const regions = ["BR", "CA", "DE", "FR", "GB", "PT", "TW", "US"];

describe("suggestPlace", () => {
  test.each([
    [["en-GB", "en"], { language: "en-GB", region: "GB" }],
    [["fr-CA"], { language: "fr-CA", region: "CA" }],
    [["fr"], { language: "fr-FR", region: "FR" }],
    [["de", "en-US"], { language: "en-US", region: "US" }],
    [["zh-Hant-TW"], { language: "zh-TW", region: "TW" }],
    [["pt-PT", "pt-BR"], { language: "pt-BR", region: "PT" }],
  ])("%j suggests %j", (tags, suggestion) => {
    expect(suggestPlace(tags, languages, regions)).toEqual(suggestion);
  });

  test("a region TMDB has no translation for still sets the region", () => {
    expect(suggestPlace(["en-DE"], languages, regions)).toEqual({ region: "DE" });
  });

  test("a language SlipStream doesn't offer suggests nothing", () => {
    expect(suggestPlace(["eo"], languages, regions)).toEqual({});
  });
});
