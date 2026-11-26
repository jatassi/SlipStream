import { HttpResponse, type RequestHandler, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { PlaybackGroup, PlaybackPlayer, PlaybackReport } from "../../api/gen/model";
import { filmPlan } from "../../test/manual";
import { manualEntry } from "../../test/manual-entry";
import { denJellyfin, livingRoomPlex } from "../../test/media-servers";
import {
  denReport,
  emptyReport,
  offlineReport,
  olderPlaysPage,
  playsPage,
} from "../../test/playback";
import { openPlayback } from "../../test/playback-page";

function open(server = denJellyfin, report?: PlaybackReport, extra: RequestHandler[] = []) {
  openPlayback(server, report, extra);
  return page.getByRole("list", { name: `Players on ${server.name}` });
}

const playerRow = (players: ReturnType<typeof open>, name: string) =>
  players.getByRole("listitem").filter({ hasText: name });

async function openDetails(name: string, server = denJellyfin, report?: PlaybackReport) {
  const row = playerRow(open(server, report), name);
  await userEvent.click(row.getByRole("button", { name: "Details" }));
  return row;
}

const onePlayer = (over: Partial<PlaybackPlayer>): PlaybackReport => {
  const [first] = denReport.players;
  if (first === undefined) {
    throw new Error("Den Jellyfin's report has a player");
  }
  return { ...denReport, players: [{ ...first, ...over }] };
};

const unknownGroup: PlaybackGroup = {
  dimensions: [{ dimension: "unknown", value: "", choosable: false }],
  cause: [{ dimension: "unknown", value: "", choosable: false }],
  reasons: [],
  targets: [],
  plays: 2,
  outcomes: [{ kind: "no_target", plays: 2 }],
};

const yesterday = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();

const reportLines: {
  name: string;
  server?: typeof denJellyfin;
  report?: PlaybackReport;
  lines: RegExp[];
}[] = [
  {
    name: "the server line counts the window's transcodes",
    lines: [/^Den Jellyfin transcoded 12 of 17 plays in the last 30 days\.$/u],
  },
  {
    name: "the window names where history starts when that is later",
    report: { ...denReport, since: yesterday },
    lines: [/^Den Jellyfin transcoded 12 of 17 plays since yesterday/u],
  },
  {
    name: "a server with no transcodes says so",
    report: { ...denReport, transcoded: 0, players: denReport.players.slice(1) },
    lines: [/^None of the 17 plays on Den Jellyfin in the last 30 days were transcoded\.$/u],
  },
  {
    name: "an empty window",
    report: emptyReport,
    lines: [/^Nothing has played on Den Jellyfin in the last 30 days/u],
  },
  {
    name: "an offline reaching Node is named",
    report: offlineReport,
    lines: [/Den Jellyfin while home is offline/u, /^Nothing has played on Den Jellyfin/u],
  },
  {
    name: "the lead names the largest group's cause and version, and counts the rest",
    lines: [
      /^9 of 15 plays on Living Room TV were transcoded because the player can't show Dolby Vision or HDR10\. The Bluray-1080p version .+ Choose it .+ Living Room TV\. 2 more plays on Living Room TV/u,
      /^None of Bedroom iPad's 2 plays were transcoded\.$/u,
    ],
  },
  {
    name: "a player whose transcodes all lack a snapshot",
    report: onePlayer({ transcoded: 4, unmatched: 4, groups: [] }),
    lines: [/^4 plays on Living Room TV were transcoded\. .+Check Path mappings\.$/u],
  },
  {
    name: "a Jellyfin play with no reason names the server",
    report: onePlayer({ transcoded: 2, groups: [unknownGroup] }),
    lines: [
      /^2 of 15 plays on Living Room TV were transcoded\. Den Jellyfin gave no reason.+quality setting\.$/u,
    ],
  },
  {
    name: "a Plex play with no reason",
    server: livingRoomPlex,
    report: { ...onePlayer({ transcoded: 2, groups: [unknownGroup] }), plex: true },
    lines: [
      /^2 of 15 plays on Living Room TV were transcoded, and what the player received.+quality setting\.$/u,
    ],
  },
  {
    name: "a Plex suggestion",
    server: livingRoomPlex,
    lines: [
      /^3 of 4 plays on Kitchen Roku were transcoded because the player can't play x265 video\. A 1080p x264 for Kitchen Roku version/u,
    ],
  },
];

test.each(reportLines)("$name", async ({ server = denJellyfin, report, lines }) => {
  open(server, report);
  await Promise.all(lines.map((line) => expect.element(page.getByText(line)).toBeVisible()));
});

const detailLines: { outcome: string; lines: RegExp[] }[] = [
  {
    outcome: "counts",
    lines: [/^Direct play 3 · Direct stream 0 · Transcode 12$/u, /^1 play was of files outside/u],
  },
  { outcome: "version", lines: [/^4 of 15 plays .+Bluray-1080p version/u] },
  {
    outcome: "suggest",
    lines: [
      /^3 of 15 plays .+A 1080p SDR for Living Room TV version would avoid that/u,
      /^Left out because they blocked direct play: Dolby Vision and HDR10\.$/u,
      /^Left out because Living Room TV hasn't direct played them in the last 30 days: DTS-HD MA\.$/u,
      /^1080p video$/u,
    ],
  },
  {
    outcome: "existing",
    lines: [
      /^2 of 15 plays .+The 1080p Target would avoid that, and 4 films played on Living Room TV need a file for it/u,
      /^Heat has no file for it yet/u,
      /^Add it to the Ultra Profile so that Ronin gets a file/u,
      /^Low Tides has a file for it, but Den Jellyfin doesn't list it yet\.$/u,
      /^The Lantern has a file for it, but Living Room TV hasn't direct played its High 10 video profile yet\.$/u,
    ],
  },
  {
    outcome: "unproven",
    lines: [
      /^1 of 15 plays .+TrueHD audio\. .+hasn't direct played audio in another codec in the last 30 days/u,
    ],
  },
  {
    outcome: "no_target",
    lines: [
      /^1 of 15 plays .+direct play failed on the player\. No Target fixes this\. Update the player app/u,
    ],
  },
  {
    outcome: "machinery",
    lines: [
      /^Reasons from Den Jellyfin: VideoRangeTypeNotSupported$/u,
      /^The transcoded files met these Targets when they played: /u,
      /^The report covers plays since .+30 days/u,
    ],
  },
];

test.each(detailLines)("Details read every $outcome line", async ({ lines }) => {
  const row = await openDetails("Living Room TV");
  await Promise.all(
    lines.map((line) =>
      expect.poll(() => row.getByText(line).elements().length).toBeGreaterThan(0),
    ),
  );
});

test("a suggestion's Details keep the Plex note and Add this Target", async () => {
  const row = await openDetails("Kitchen Roku", livingRoomPlex);
  await expect.element(row.getByText(/^Plex gives no reasons/u)).toBeVisible();
  await expect.element(row.getByRole("link", { name: "Add this Target" }).first()).toBeVisible();
});

test("Search for one opens Manual search on the Work's Item", async () => {
  const manual = manualEntry(filmPlan);
  const row = playerRow(open(denJellyfin, undefined, manual.handlers), "Living Room TV");
  await userEvent.click(row.getByRole("button", { name: "Details" }));
  const heat = row.getByRole("listitem").filter({ hasText: /^Heat has no file/u });
  await userEvent.click(heat.getByRole("button", { name: "Search for one" }));
  const existing = denReport.players[0]?.groups[0]?.outcomes.find(
    ({ kind }) => kind === "existing",
  );
  await expect.poll(() => manual.started).toEqual([{ itemIds: [existing?.works?.[0]?.itemId] }]);
});

test("Show plays lists the player's plays newest first and pages to older ones", async () => {
  const asked: (string | null)[] = [];
  const answer = http.get("/api/v1/media-servers/:id/plays", ({ request }) => {
    const url = new URL(request.url);
    asked.push(url.searchParams.get("player"));
    return HttpResponse.json(
      url.searchParams.get("cursor") === playsPage.next ? olderPlaysPage : playsPage,
    );
  });
  const row = playerRow(open(denJellyfin, undefined, [answer]), "Living Room TV");
  await userEvent.click(row.getByRole("button", { name: "Details" }));
  await userEvent.click(row.getByRole("button", { name: "Show plays" }));
  const plays = page.getByRole("dialog", { name: "Plays on Living Room TV" }).getByRole("list");
  await expect
    .element(plays.getByRole("listitem").nth(1))
    .toMatchTextContent(/Heat \(1995\)\.mp4/u);
  await expect
    .element(plays.getByRole("listitem").first())
    .toMatchTextContent(/The Quiet Orchard/u);
  await userEvent.click(page.getByRole("button", { name: "Show older plays" }));
  await expect.element(plays.getByRole("listitem")).toHaveLength(3);
  expect(asked).toEqual(["lrtv", "lrtv"]);
});
