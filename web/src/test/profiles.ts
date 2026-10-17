import type { TargetsCutList, TargetsProfile, TargetsProfileDetail } from "../api/gen/model";
import {
  getGetProfileMockHandler,
  getListCutsMockHandler,
  getListProfilesMockHandler,
} from "../api/gen/query/targets/targets.msw";
import { keeper, starter } from "./targets";

const cut = (key: string, name: string, aliases: string[]) => ({
  key,
  name,
  aliases,
  origin: "builtin" as const,
});

/** GET /cuts: the built-in cuts, then one the cinema test type adds and one a zine type adds. */
export const cutList: TargetsCutList = {
  cuts: [
    cut("standard", "Standard", []),
    cut("theatrical", "Theatrical", ["Theatrical"]),
    cut("extended", "Extended", ["Extended", "Extended Cut"]),
    cut("directors-cut", "Director's Cut", ["Director's Cut"]),
    cut("imax", "IMAX", ["IMAX"]),
    {
      key: "superfan",
      name: "Superfan",
      aliases: ["Superfan"],
      origin: "type",
      contentType: "cinema",
    },
    { key: "deluxe", name: "Deluxe", aliases: ["Deluxe"], origin: "type", contentType: "zine" },
  ],
};

/** The cinema test type's default Profile, holding the starter Target. */
export const films: TargetsProfile = {
  id: "0199e1f2-7a3b-7c4d-8e5f-000000000001",
  name: "Films",
  contentType: "cinema",
  default: true,
  targets: [{ targetId: starter.id, sourcing: "download" }],
  editions: { kind: "any" },
  works: 12,
};

/** A second cinema Profile, wanting 2160p HDR first and the starter after it. */
export const ultra: TargetsProfile = {
  id: "0199e1f2-7a3b-7c4d-8e5f-000000000002",
  name: "Ultra",
  contentType: "cinema",
  default: false,
  targets: [
    { targetId: keeper.id, sourcing: "download" },
    { targetId: starter.id, sourcing: "download" },
  ],
  editions: { kind: "any" },
  works: 1,
};

/** Ultra's detail: the starter is estimated from its files, 2160p HDR has none yet. */
export const ultraDetail: TargetsProfileDetail = {
  ...ultra,
  estimates: [
    { targetId: keeper.id, mibPerMinute: null, bytes: null },
    { targetId: starter.id, mibPerMinute: 20.04, bytes: 2_202_009_600 },
  ],
};

export const profileHandlers = (profiles: readonly TargetsProfile[] = [films, ultra]) => [
  getListCutsMockHandler(cutList),
  getListProfilesMockHandler({ profiles: [...profiles] }),
  getGetProfileMockHandler(({ params }) => {
    const profile = profiles.find((candidate) => candidate.id === params["profileId"]);
    if (profile === undefined) {
      throw new Error(`no Profile ${String(params["profileId"])}`);
    }
    return { ...profile, estimates: profile.id === ultra.id ? ultraDetail.estimates : [] };
  }),
];
