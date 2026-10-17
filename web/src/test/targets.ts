import { HttpResponse, http } from "msw";

import type { TargetsQualityList, TargetsTarget } from "../api/gen/model";
import {
  getListQualitiesMockHandler,
  getListTargetsMockHandler,
} from "../api/gen/query/targets/targets.msw";

const quality = (
  key: TargetsQualityList["qualities"][number]["key"],
  name: string,
  source: TargetsQualityList["qualities"][number]["source"],
  resolution: TargetsQualityList["qualities"][number]["resolution"],
  floor: number,
) => ({ key, name, source, resolution, cinema: source === "cinema", floor });

/** GET /qualities as the Node answers it. */
export const qualityList: TargetsQualityList = {
  qualities: [
    quality("sdtv", "SDTV", "tv", "480p", 0),
    quality("dvd", "DVD", "dvd", "480p", 0),
    quality("webrip-480p", "WEBRip-480p", "webrip", "480p", 0),
    quality("hdtv-720p", "HDTV-720p", "hdtv", "720p", 17.1),
    quality("webrip-720p", "WEBRip-720p", "webrip", "720p", 12.5),
    quality("webdl-720p", "WEBDL-720p", "webdl", "720p", 12.5),
    quality("bluray-720p", "Bluray-720p", "bluray", "720p", 25.7),
    quality("hdtv-1080p", "HDTV-1080p", "hdtv", "1080p", 33.8),
    quality("webrip-1080p", "WEBRip-1080p", "webrip", "1080p", 12.5),
    quality("webdl-1080p", "WEBDL-1080p", "webdl", "1080p", 12.5),
    quality("bluray-1080p", "Bluray-1080p", "bluray", "1080p", 50.8),
    quality("remux-1080p", "Remux-1080p", "remux", "1080p", 102),
    quality("hdtv-2160p", "HDTV-2160p", "hdtv", "2160p", 85),
    quality("webrip-2160p", "WEBRip-2160p", "webrip", "2160p", 34.5),
    quality("webdl-2160p", "WEBDL-2160p", "webdl", "2160p", 34.5),
    quality("bluray-2160p", "Bluray-2160p", "bluray", "2160p", 102),
    quality("remux-2160p", "Remux-2160p", "remux", "2160p", 187.4),
    quality("cam", "CAM", "cinema", null, 0),
  ],
  presets: [
    {
      key: "disc-or-resolution",
      name: "Upgrade for disc or resolution",
      tiers: [
        ["bluray-2160p", "remux-2160p"],
        ["hdtv-2160p", "webrip-2160p", "webdl-2160p"],
        ["bluray-1080p", "remux-1080p"],
        ["hdtv-1080p", "webrip-1080p", "webdl-1080p"],
        ["bluray-720p"],
        ["hdtv-720p", "webrip-720p", "webdl-720p"],
        ["dvd"],
        ["sdtv", "webrip-480p"],
        ["cam"],
      ],
    },
    {
      key: "anything-better",
      name: "Upgrade for anything better",
      tiers: [
        ["remux-2160p"],
        ["bluray-2160p"],
        ["webdl-2160p"],
        ["webrip-2160p"],
        ["hdtv-2160p"],
        ["remux-1080p"],
        ["bluray-1080p"],
        ["webdl-1080p"],
        ["webrip-1080p"],
        ["hdtv-1080p"],
        ["bluray-720p"],
        ["webdl-720p"],
        ["webrip-720p"],
        ["hdtv-720p"],
        ["webrip-480p"],
        ["dvd"],
        ["sdtv"],
        ["cam"],
      ],
    },
    {
      key: "resolution-only",
      name: "Upgrade for resolution only",
      tiers: [
        ["hdtv-2160p", "webrip-2160p", "webdl-2160p", "bluray-2160p", "remux-2160p"],
        ["hdtv-1080p", "webrip-1080p", "webdl-1080p", "bluray-1080p", "remux-1080p"],
        ["hdtv-720p", "webrip-720p", "webdl-720p", "bluray-720p"],
        ["sdtv", "dvd", "webrip-480p"],
        ["cam"],
      ],
    },
  ],
};

/** The starter Target a fresh Controller makes, held by the Movies Profile. */
export const starter: TargetsTarget = {
  id: "0199e1f2-7a3b-7c4d-8e5f-6a7b8c9d0e1f",
  name: "1080p",
  createdAt: "2026-10-17T09:00:00Z",
  tiers: [
    ["bluray-1080p", "remux-1080p"],
    ["hdtv-1080p", "webrip-1080p", "webdl-1080p"],
    ["bluray-720p"],
    ["hdtv-720p", "webrip-720p", "webdl-720p"],
  ],
  cutoff: "bluray-1080p",
  filters: [],
  sizes: [
    { quality: "bluray-1080p", min: 50.8, max: null },
    { quality: "remux-1080p", min: 102, max: null },
    { quality: "hdtv-1080p", min: 33.8, max: null },
    { quality: "webrip-1080p", min: 12.5, max: null },
    { quality: "webdl-1080p", min: 12.5, max: 40 },
    { quality: "bluray-720p", min: 25.7, max: null },
    { quality: "hdtv-720p", min: 17.1, max: null },
    { quality: "webrip-720p", min: 12.5, max: null },
    { quality: "webdl-720p", min: 12.5, max: null },
  ],
  profiles: [{ id: "0199e1f2-7a3b-7c4d-8e5f-000000000001", name: "Movies" }],
  contains: [],
};

/** A Target no Profile holds, that never upgrades and only takes HDR10. */
export const keeper: TargetsTarget = {
  id: "0199e1f2-7a3b-7c4d-8e5f-6a7b8c9d0e2a",
  name: "2160p HDR",
  createdAt: "2026-10-17T09:05:00Z",
  tiers: [["remux-2160p"], ["webdl-2160p", "webrip-2160p"]],
  cutoff: "webdl-2160p",
  filters: [{ attribute: "hdr", mode: "only", values: ["HDR10"] }],
  sizes: [
    { quality: "remux-2160p", min: 187.4, max: null },
    { quality: "webdl-2160p", min: 34.5, max: null },
    { quality: "webrip-2160p", min: 34.5, max: null },
  ],
  profiles: [],
  contains: [],
};

export const targetHandlers = (targets: readonly TargetsTarget[] = [starter, keeper]) => [
  getListQualitiesMockHandler(qualityList),
  getListTargetsMockHandler({ targets: [...targets] }),
  ...targets.map((target) =>
    http.get(`*/api/v1/targets/${target.id}`, () => HttpResponse.json(target)),
  ),
];
