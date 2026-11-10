import type {
  GrabsChoice,
  GrabsManualSearch,
  GrabsPlanRow,
  GrabsReleaseDetails,
} from "../api/gen/model";
import { longShelf } from "./anthology";
import { prowlarrId, torrentLeechId } from "./indexers";
import { starter } from "./targets";

export const manualSearchId = "0199f3a1-5c2e-7d4f-8a6b-1c3e5a7c9e01";
export const runId = "0199f3a1-5c2e-7d4f-8a6b-1c3e5a7c9e02";
const seasonId = "0199f3a1-5c2e-7d4f-8a6b-1c3e5a7c9e03";
export const heldGrabId = "0199f3a1-5c2e-7d4f-8a6b-1c3e5a7c9e04";

const [storyOne, storyTwo, storyThree] = longShelf.items;

export const release = (
  id: string,
  title: string,
  words: string,
  part: Partial<GrabsReleaseDetails> = {},
): GrabsReleaseDetails => ({
  id,
  title,
  words,
  indexers: ["Prowlarr"],
  size: 4_294_967_296,
  publishedAt: "2026-11-01T12:00:00Z",
  seeders: 40,
  freeleech: false,
  formatIds: [],
  verdicts: ["It would fill 1080p."],
  lostAt: null,
  blocked: null,
  ...part,
});

export const flux = release(
  `${runId}.0`,
  "The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.H.264-FLUX",
  "WEBDL-1080p · DD+ 5.1 · FLUX · 4.0 GB",
);
export const lowScore = release(
  `${runId}.3`,
  "The.Quiet.Orchard.2021.1080p.WEBRip.x264-RARBG",
  "WEBRip-1080p · RARBG · 2.1 GB",
  { indexers: ["TorrentLeech"], seeders: 9, size: 2_254_857_830 },
);
export const blocked = release(
  `${runId}.5`,
  "The.Quiet.Orchard.2021.1080p.BluRay.x264-SPARKS",
  "Bluray-1080p · SPARKS · 9.8 GB",
  {
    size: 10_522_669_875,
    blocked:
      "It's been on the Blocklist since 2 Nov because TorrentLeech reported it unregistered.",
  },
);

export const lowScoreChoice: GrabsChoice = {
  pin: "pin-orchard-1080p-rarbg",
  cells: ["cell-orchard-1080p"],
  release: lowScore,
  lifts: [
    {
      code: "score_below_minimum",
      targetId: starter.id,
      value: "5",
      soft: true,
      words: "Its score of 5 is under 1080p's minimum of 10.",
    },
  ],
  edition: null,
  note: "From TorrentLeech, which automatic search doesn't use.",
};

/** A blocked choice: Other choices shows since when and why, with "Remove from Blocklist and grab". */
export const blockedChoice: GrabsChoice = {
  pin: "pin-orchard-1080p-sparks",
  cells: ["cell-orchard-1080p"],
  release: blocked,
  lifts: [],
  edition: null,
};

const orchardRow: GrabsPlanRow = {
  key: "cell-orchard-1080p",
  pick: {
    release: flux,
    cells: ["cell-orchard-1080p"],
    lifts: [],
    edition: null,
    routed: "qBittorrent on Den",
  },
  pinned: false,
  outcome: "It gets you 1080p for The Quiet Orchard (2021).",
  choices: [lowScoreChoice, blockedChoice],
};

const orchardGap: GrabsPlanRow = {
  key: "cell-orchard-2160p",
  pick: null,
  pinned: false,
  missing: "Missing: 2160p HDR for The Quiet Orchard (2021).",
  cause: "No Indexer listed a 2160p Release.",
  choices: [],
};

/** A film's plan: one Grab, a 2160p HDR cell nothing fills, two other choices and the folded line. */
export const filmPlan: GrabsManualSearch = {
  id: manualSearchId,
  scope: { levelIds: [], itemIds: [] },
  title: "The Quiet Orchard (2021)",
  query: null,
  startedAt: "2026-11-10T14:02:00Z",
  startedBy: "Ada Okafor",
  searching: false,
  queryable: true,
  line: "SlipStream searched 2 Indexers.",
  indexers: [
    {
      indexerId: prowlarrId,
      name: "Prowlarr",
      sentence: "Found 6 Releases",
      done: true,
      automatic: true,
      failed: false,
    },
    {
      indexerId: torrentLeechId,
      name: "TorrentLeech",
      sentence: "Found 3 Releases",
      done: true,
      automatic: false,
      failed: false,
    },
  ],
  plan: {
    headline:
      "1 Grab gets you 1 of the 2 files you're missing. SlipStream hasn't found a Release for 2160p HDR yet.",
    rows: [orchardRow, orchardGap],
    held: [],
  },
  folded: {
    line: "SlipStream wouldn't grab 4 more Releases.",
    groups: [
      { words: "3 are 720p, which no Target here takes.", fault: false },
      { words: "1 is on the Blocklist.", fault: false },
    ],
  },
  releaseCount: 9,
  pins: [],
};

/** The film's plan once the low-score choice is pinned in place of FLUX. */
export const pinnedFilmPlan: GrabsManualSearch = {
  ...filmPlan,
  plan: {
    ...filmPlan.plan,
    rows: [
      {
        ...orchardRow,
        pinned: true,
        pick: {
          release: lowScore,
          cells: [orchardRow.key],
          lifts: lowScoreChoice.lifts,
          edition: null,
          routed: "qBittorrent on Den",
        },
        choices: [{ ...lowScoreChoice, pin: "pin-orchard-1080p-flux", release: flux, lifts: [] }],
      },
      orchardGap,
    ],
  },
  pins: [{ key: orchardRow.key, pin: lowScoreChoice.pin }],
};

const story = (position: number, title: string): GrabsReleaseDetails =>
  release(`${seasonId}.${position}`, title, "WEBDL-1080p · NTb · 1.2 GB", {
    size: 1_288_490_189,
    verdicts: ["It would fill 1080p.", "2160p HDR refuses it because its quality isn't in it."],
  });

/** A season's plan, still searching: two Grabs, a cell nothing fills and a cell a Grab downloads. */
export const seasonPlan: GrabsManualSearch = {
  id: manualSearchId,
  scope: { levelIds: ["0199f3a1-5c2e-7d4f-8a6b-1c3e5a7c9eb0"], itemIds: [] },
  title: "Volume 1 of The Long Shelf",
  query: null,
  startedAt: "2026-11-10T14:05:00Z",
  startedBy: "the API key",
  searching: true,
  queryable: false,
  line: "Searching 2 Indexers. 1 has responded.",
  indexers: [
    {
      indexerId: prowlarrId,
      name: "Prowlarr",
      sentence: "Found 14 Releases",
      done: true,
      automatic: true,
      failed: false,
    },
    {
      indexerId: torrentLeechId,
      name: "TorrentLeech",
      sentence: "Searching",
      done: false,
      automatic: true,
      failed: false,
    },
  ],
  plan: {
    headline:
      "2 Grabs get you 2 of the 3 files you're missing. SlipStream hasn't found a Release for S03 yet.",
    rows: [
      {
        key: `cell-${storyOne?.id}`,
        pick: {
          release: story(0, "The.Long.Shelf.V01S01.1080p.WEB-DL-NTb"),
          cells: [`cell-${storyOne?.id}`],
          lifts: [],
          edition: null,
          routed: "the built-in client on home",
        },
        pinned: false,
        outcome: "It gets you 1080p for S01.",
        choices: [],
      },
      {
        key: `cell-${storyTwo?.id}`,
        pick: {
          release: story(1, "The.Long.Shelf.V01S02.1080p.WEB-DL-NTb"),
          cells: [`cell-${storyTwo?.id}`],
          lifts: [],
          edition: null,
          routed: "the built-in client on home",
        },
        pinned: false,
        outcome: "It gets you 1080p for S02.",
        choices: [],
      },
      {
        key: `cell-${storyThree?.id}`,
        pick: null,
        pinned: false,
        missing: "Missing: 1080p for S03.",
        cause: "No Indexer listed a 1080p Release.",
        choices: [],
      },
    ],
    held: [
      {
        grabId: heldGrabId,
        words: "A Grab is downloading The.Long.Shelf.V01S04.1080p.WEB-DL-NTb for 1080p.",
        stage: "downloading",
      },
    ],
  },
  folded: { line: "SlipStream wouldn't grab 3 more Releases.", groups: [] },
  releaseCount: 17,
  pins: [],
};
