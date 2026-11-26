import { HttpResponse, http } from "msw";

import type {
  MediaserversPlayView,
  MediaserversPlays,
  PlaybackBlocked,
  PlaybackEvidence,
  PlaybackOutcome,
  PlaybackPlayer,
  PlaybackReport,
  PlaybackSuggestion,
  PlaybackTargetRef,
  PlaybackWorkLine,
} from "../api/gen/model";
import { orchardId } from "./library";
import { denJellyfin, livingRoomPlex } from "./media-servers";
import { ultra } from "./profiles";
import { keeper, starter } from "./targets";

const starterRef: PlaybackTargetRef = { id: starter.id, name: starter.name };
const keeperRef: PlaybackTargetRef = { id: keeper.id, name: keeper.name };

const blocked = (
  dimension: PlaybackBlocked["dimension"],
  value: string,
  choosable = true,
): PlaybackBlocked => ({ dimension, value, choosable });

const dolbyVision = blocked("hdr", "DV");
const hdr10 = blocked("hdr", "HDR10");
const trueHD = blocked("audio_codec", "TrueHD");
const x265 = blocked("video_codec", "x265");

const proven = (fact: PlaybackEvidence["fact"], value: string, n: number): PlaybackEvidence => ({
  fact,
  value,
  playId: `0199f3c2-4444-7000-8000-0000000000${n}`,
  workId: orchardId,
  title: "The Quiet Orchard",
  contentType: "cinema",
  startedAt: "2026-11-12T19:00:00Z",
});

const sdrEvidence = [proven("resolution", "1080p", 10), proven("hdr", "SDR", 10)];

const workLine = (
  n: number,
  title: string,
  state: PlaybackWorkLine["state"],
  more: Partial<PlaybackWorkLine> = {},
): PlaybackWorkLine => ({
  workId: `0199f3c3-5555-7000-8000-00000000000${n}`,
  itemId: `0199f3c3-5555-7000-8000-00000000001${n}`,
  title,
  contentType: "cinema",
  state,
  ...more,
});

const group = (dimensions: PlaybackBlocked[], reasons: string[], outcomes: PlaybackOutcome[]) => ({
  dimensions,
  cause: dimensions,
  reasons,
  targets: [keeperRef],
  plays: outcomes.reduce((n, o) => n + o.plays, 0),
  outcomes,
});

const suggestion = (over: Partial<PlaybackSuggestion>): PlaybackSuggestion => ({
  name: "",
  highest: "1080p",
  videoCodecs: [],
  notHdr: [],
  notAudioCodecs: [],
  notChannels: [],
  blocked: [],
  unproven: [],
  ...over,
});

const player = (
  over: Partial<PlaybackPlayer> & Pick<PlaybackPlayer, "key" | "name">,
): PlaybackPlayer => {
  const counts = { directPlays: 0, directStreams: 0, transcoded: 0, unmatched: 0, ...over };
  return {
    product: "Jellyfin Android TV",
    platform: "AndroidTv",
    lastPlayedAt: "2026-11-14T20:00:00Z",
    plays: counts.directPlays + counts.directStreams + counts.transcoded,
    groups: [],
    ...counts,
  };
};

const sum = (players: PlaybackPlayer[], count: "plays" | "transcoded" | "unmatched") =>
  players.reduce((n, p) => n + p[count], 0);

const report = (players: PlaybackPlayer[], over: Partial<PlaybackReport> = {}) => ({
  since: "2026-10-15T20:00:00Z",
  window: 30 * 24 * 60 * 60,
  reachable: true,
  plex: false,
  plays: sum(players, "plays"),
  transcoded: sum(players, "transcoded"),
  unmatched: sum(players, "unmatched"),
  players,
  ...over,
});

/**
 * Den Jellyfin's report: Living Room TV transcodes Dolby Vision, with every
 * outcome kind and every Work state, and Bedroom iPad plays everything directly.
 */
export const denReport: PlaybackReport = report([
  player({
    key: "lrtv",
    name: "Living Room TV",
    directPlays: 3,
    transcoded: 12,
    unmatched: 1,
    groups: [
      group(
        [dolbyVision, hdr10],
        ["VideoRangeTypeNotSupported"],
        [
          {
            kind: "version",
            plays: 4,
            version: {
              fileId: "0199f3c4-6666-7000-8000-000000000001",
              label: "Bluray-1080p",
              separateItem: false,
            },
            evidence: sdrEvidence,
          },
          {
            kind: "suggest",
            plays: 3,
            suggestion: suggestion({
              name: "1080p SDR for Living Room TV",
              videoCodecs: ["x264", "x265"],
              notHdr: ["DV", "HDR10"],
              notAudioCodecs: ["DTS-HD MA"],
              blocked: [dolbyVision, hdr10],
              unproven: [blocked("audio_codec", "DTS-HD MA")],
            }),
            evidence: sdrEvidence,
          },
          {
            kind: "existing",
            plays: 2,
            target: starterRef,
            works: [
              workLine(1, "Heat", "no_file"),
              workLine(2, "Ronin", "not_in_profile", {
                profile: { id: ultra.id, name: ultra.name },
              }),
              workLine(3, "Low Tides", "not_listed"),
              workLine(4, "The Lantern", "not_proven", {
                fact: { fact: "video_profile", value: "High 10" },
              }),
            ],
            evidence: sdrEvidence,
          },
        ],
      ),
      group(
        [trueHD],
        ["AudioCodecNotSupported"],
        [{ kind: "unproven", plays: 1, missing: trueHD }],
      ),
      group(
        [blocked("player_error", "", false)],
        ["DirectPlayError"],
        [{ kind: "no_target", plays: 1 }],
      ),
    ],
  }),
  player({
    key: "bedroom-ipad",
    name: "Bedroom iPad",
    product: "Jellyfin Mobile",
    platform: "iOS",
    lastPlayedAt: "2026-11-13T21:30:00Z",
    directPlays: 2,
  }),
]);

/** Den Jellyfin's report while its Connect from Node is offline. */
export const offlineReport: PlaybackReport = report([], { reachable: false });

/** A report with no plays in the window. */
export const emptyReport: PlaybackReport = report([]);

/** Living room Plex's report, which has no reasons: Kitchen Roku transcodes HEVC. */
export const plexReport: PlaybackReport = report(
  [
    player({
      key: "8f2c0a1e9d7b",
      name: "Kitchen Roku",
      product: "Plex for Roku",
      platform: "Roku",
      directPlays: 1,
      transcoded: 3,
      groups: [
        group(
          [x265],
          [],
          [
            {
              kind: "suggest",
              plays: 3,
              suggestion: suggestion({
                name: "1080p x264 for Kitchen Roku",
                videoCodecs: ["x264"],
                blocked: [x265],
              }),
              evidence: [proven("video_codec", "x264", 20)],
            },
          ],
        ),
      ],
    }),
  ],
  { plex: true },
);

const unreported = {
  container: null,
  videoCodec: null,
  height: null,
  bitrate: null,
  audioCodec: null,
  channels: null,
};

const directPlay = (n: number, startedAt: string, path: string): MediaserversPlayView => ({
  id: `0199f3c2-4444-7000-8000-00000000010${n}`,
  playerKey: "lrtv",
  player: "Living Room TV",
  product: "Jellyfin Android TV",
  platform: "AndroidTv",
  user: "grace",
  path,
  decision: "direct_play",
  video: "direct",
  audio: "direct",
  subtitles: "none",
  hardware: false,
  reasons: [],
  received: unreported,
  startedAt,
  lastSeenAt: startedAt,
  endedAt: startedAt,
});

const orchard = "/data/films/The Quiet Orchard (2021)/The Quiet Orchard (2021)";

/** Living Room TV's newest plays: an open transcode and a direct play with no snapshot. */
export const playsPage: MediaserversPlays = {
  plays: [
    {
      ...directPlay(1, "2026-11-14T20:00:00Z", `${orchard} Remux-2160p.mkv`),
      work: { workId: orchardId, title: "The Quiet Orchard", label: "Remux-2160p" },
      decision: "transcode",
      video: "transcode",
      hardware: true,
      reasons: ["VideoRangeTypeNotSupported"],
      received: { ...unreported, container: "ts", videoCodec: "h264", height: 1080 },
      lastSeenAt: "2026-11-14T20:45:00Z",
      endedAt: null,
    },
    directPlay(2, "2026-11-13T19:00:00Z", "/data/films/Heat (1995)/Heat (1995).mp4"),
  ],
  next: "0199f3c2-4444-7000-8000-000000000102",
};

/** The page after playsPage, the last. */
export const olderPlaysPage: MediaserversPlays = {
  plays: [directPlay(3, "2026-11-12T19:00:00Z", `${orchard} Bluray-1080p.mkv`)],
};

const reportsById: Record<string, PlaybackReport> = {
  [denJellyfin.id]: denReport,
  [livingRoomPlex.id]: plexReport,
};

/** Answers each server's Playback report, and its plays two pages deep. */
export function playbackHandlers(reports = reportsById) {
  return [
    http.get<{ id: string }>("/api/v1/media-servers/:id/playback", ({ params }) =>
      HttpResponse.json(reports[params.id]),
    ),
    http.get<{ id: string }>("/api/v1/media-servers/:id/plays", ({ request }) => {
      const older = new URL(request.url).searchParams.get("cursor") === playsPage.next;
      return HttpResponse.json(older ? olderPlaysPage : playsPage);
    }),
  ];
}
