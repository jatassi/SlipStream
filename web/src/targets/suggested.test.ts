import { expect, test } from "vitest";

import type { PlaybackSuggestion, TargetsNewTarget } from "../api/gen/model";
import { qualityList } from "../test/targets";
import { bodyOf, newDraft } from "./draft";
import { newTargetSearch, suggestedDraft } from "./suggested";

const suggestion = (over: Partial<PlaybackSuggestion>): PlaybackSuggestion => ({
  name: "1080p HDR10",
  highest: "1080p",
  videoCodecs: ["x265"],
  notHdr: [],
  notAudioCodecs: [],
  notChannels: [],
  blocked: [],
  unproven: [],
  ...over,
});

const created = (s: PlaybackSuggestion, cinema = false) =>
  bodyOf(suggestedDraft({ ...newDraft(qualityList, []), cinema }, s, qualityList.qualities));

type Expected = Pick<TargetsNewTarget, "name" | "tiers" | "cutoff" | "filters">;

const at1080p = [
  ["bluray-1080p", "remux-1080p"],
  ["hdtv-1080p", "webrip-1080p", "webdl-1080p"],
  ["bluray-720p"],
  ["hdtv-720p", "webrip-720p", "webdl-720p"],
  ["dvd"],
  ["sdtv", "webrip-480p"],
];

test.each<[string, PlaybackSuggestion, boolean, Expected]>([
  [
    "Qualities above 1080p dropped, the cutoff on the best remaining tier, every filter set",
    suggestion({
      notHdr: ["DV"],
      notAudioCodecs: ["TrueHD", "DTS-HD MA"],
      notChannels: ["7.1"],
    }),
    false,
    {
      name: "1080p HDR10",
      tiers: at1080p,
      cutoff: "bluray-1080p",
      filters: [
        { attribute: "hdr", mode: "not", values: ["DV"] },
        { attribute: "video_codec", mode: "only", values: ["x265"] },
        { attribute: "audio_codec", mode: "not", values: ["TrueHD", "DTS-HD MA"] },
        { attribute: "channels", mode: "not", values: ["7.1"] },
      ],
    },
  ],
  [
    "2160p keeps every tier, and an empty Not stays Any",
    suggestion({ name: "2160p DV", highest: "2160p", videoCodecs: ["x264", "x265"] }),
    false,
    {
      name: "2160p DV",
      tiers: [
        ["bluray-2160p", "remux-2160p"],
        ["hdtv-2160p", "webrip-2160p", "webdl-2160p"],
        ...at1080p,
      ],
      cutoff: "bluray-2160p",
      filters: [{ attribute: "video_codec", mode: "only", values: ["x264", "x265"] }],
    },
  ],
  [
    "CAM, with no resolution, is dropped from a draft that accepted it",
    suggestion({
      name: "480p SDR",
      highest: "480p",
      videoCodecs: ["XviD"],
      notHdr: ["DV", "HDR10"],
    }),
    true,
    {
      name: "480p SDR",
      tiers: [["dvd"], ["sdtv", "webrip-480p"]],
      cutoff: "dvd",
      filters: [
        { attribute: "hdr", mode: "not", values: ["DV", "HDR10"] },
        { attribute: "video_codec", mode: "only", values: ["XviD"] },
      ],
    },
  ],
])("%s", (_, s, cinema, expected) => {
  expect(created(s, cinema)).toMatchObject(expected);
});

const from = {
  server: "Living room Jellyfin",
  player: "Living Room TV",
  suggestion: suggestion({}),
};

test.each<[string, unknown]>([
  [
    "a video codec outside the vocabulary",
    { ...from, suggestion: { ...from.suggestion, videoCodecs: ["h264"] } },
  ],
  [
    "a resolution no Quality has",
    { ...from, suggestion: { ...from.suggestion, highest: "4320p" } },
  ],
  [
    "a channel layout outside the vocabulary",
    { ...from, suggestion: { ...from.suggestion, notChannels: ["9.1"] } },
  ],
  ["a server name over 200 characters", { ...from, server: "x".repeat(201) }],
  ["no suggestion", { server: from.server, player: from.player }],
  ["not an object", "1080p"],
])("a from with %s starts from the defaults", (_, bad) => {
  expect(newTargetSearch.parse({ from: bad })).toEqual({ from: undefined });
});

test("a valid from is kept", () => {
  expect(newTargetSearch.parse({ from })).toEqual({ from });
});
