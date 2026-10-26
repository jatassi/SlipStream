import { describe, expect, test } from "vitest";

import { suggestPlace } from "./language";

const languages = ["de-DE", "en-GB", "en-US", "fr-CA", "fr-FR", "ja-JP", "pt-BR", "zh-TW"];
const regions = ["BR", "CA", "DE", "FR", "GB", "JP", "PT", "TW", "US"];

describe("suggestPlace", () => {
  test.each([
    [["en-GB", "en"], { language: "en-GB", region: "GB" }],
    [["fr-CA"], { language: "fr-CA", region: "CA" }],
    [["fr"], { language: "fr-FR" }],
    [["de", "en-US", "en"], { language: "de-DE", region: "US" }],
    [["ja"], { language: "ja-JP" }],
    [["eo", "ja", "en-GB"], { language: "ja-JP", region: "GB" }],
    [["zh-Hant-TW"], { language: "zh-TW", region: "TW" }],
    [["zh-Hant"], { language: "zh-TW" }],
    [["pt-PT", "pt-BR"], { language: "pt-BR", region: "PT" }],
    [["en-DE"], { language: "en-US", region: "DE" }],
    [["en-GB", "fr-FR"], { language: "en-GB", region: "GB" }],
    [["en-NG", "fr-FR"], { language: "en-US", region: "FR" }],
  ])("%j suggests %j", (tags, suggestion) => {
    expect(suggestPlace(tags, languages, regions)).toEqual(suggestion);
  });

  test("a language SlipStream doesn't offer suggests nothing", () => {
    expect(suggestPlace(["eo"], languages, regions)).toEqual({});
  });
});
