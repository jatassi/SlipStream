import { expect, test } from "vitest";

import {
  GetManualSearchResponse,
  ListBlocklistResponse,
  ListManualSearchReleasesResponse,
} from "../api/gen/zod/grabs/grabs.zod";
import { blocklistPage } from "./blocklist";
import { filmPlan, pinnedFilmPlan, seasonPlan } from "./manual";
import { filmReleases } from "./manual-server";

test.each([
  ["the film's plan", filmPlan],
  ["the pinned film plan", pinnedFilmPlan],
  ["the season's plan", seasonPlan],
])("%s is a getManualSearch response", (_, plan) => {
  expect(GetManualSearchResponse.parse(plan)).toEqual(plan);
});

test("All Releases and the Blocklist page are their operations' responses", () => {
  expect(ListManualSearchReleasesResponse.parse({ releases: filmReleases })).toEqual({
    releases: filmReleases,
  });
  expect(ListBlocklistResponse.parse(blocklistPage)).toEqual(blocklistPage);
});
