import { HttpResponse, http } from "msw";

import type {
  MediaserversAccountView,
  MediaserversChangeView,
  MediaserversHolderView,
  MediaserversKindView,
  MediaserversLibraries,
  MediaserversPlexServerView,
  MediaserversServerView,
  MediaserversSessionView,
  SettingDurationField,
} from "../api/gen/model";
import { orchardId } from "./library";
import { home, seedbox } from "./nodes";
import { archiveRoot, filmsRoot } from "./roots";

const api = "/api/v1/media-servers";

const homeRef = { id: home.id, name: home.name };
const seedboxRef = { id: seedbox.id, name: seedbox.name };

export const kinds: MediaserversKindView[] = [
  { kind: "plex", label: "Plex", signIn: "plex_pin", browser: false, floor: "1.43" },
  { kind: "jellyfin", label: "Jellyfin", signIn: "api_key_or_admin", browser: true, floor: "12.2" },
  { kind: "emby", label: "Emby", signIn: "api_key_or_admin", browser: true, floor: "4.10" },
];

/** A healthy Jellyfin server the Controller reaches, its one shown root confirmed. */
export const denJellyfin: MediaserversServerView = {
  id: "0199f3c0-2222-7000-8000-000000000001",
  kind: "jellyfin",
  name: "Den Jellyfin",
  address: "http://192.168.1.20:8096",
  browserAddress: "http://192.168.1.20:8096",
  version: "12.2.0",
  identity: "f4c1d2e3a4b5c6d7e8f9a0b1c2d3e4f5",
  reachingNode: homeRef,
  fallback: "available",
  checkedAt: "2026-11-14T09:00:00Z",
  contact: "responding",
  contactSince: "2026-11-14T09:00:00Z",
  sends: "ok",
  lastContactAt: "2026-11-14T09:30:00Z",
  lastSendAt: "2026-11-14T09:20:00Z",
  state: { level: "ok", words: "Den Jellyfin is up to date." },
  mappingsNeeded: 0,
};

/** A Plex server reached through a Worker, unhealthy and with a root still to map. */
export const livingRoomPlex: MediaserversServerView = {
  id: "0199f3c0-2222-7000-8000-000000000002",
  kind: "plex",
  name: "Living room Plex",
  address: "https://192-168-1-30.a1b2c3.plex.direct:32400",
  version: "1.43.5.11029",
  identity: "9b8a7c6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b",
  reachingNode: seedboxRef,
  runsOnNode: seedboxRef,
  fallback: "none",
  checkedAt: "2026-11-14T08:00:00Z",
  contact: "not_responding",
  contactSince: "2026-11-14T14:02:00Z",
  sends: "failing",
  lastContactAt: "2026-11-14T14:01:00Z",
  unhealthy: { reason: "not_responding", since: "2026-11-14T14:07:00Z" },
  state: {
    level: "fail",
    words:
      "Living room Plex hasn't responded to Seedbox since 14:02. Check that Plex is running on Seedbox and that Seedbox can reach 192.168.1.30:32400.",
  },
  mappingsNeeded: 1,
  plexPass: true,
};

export const servers = [denJellyfin, livingRoomPlex];

export const denLibraries: MediaserversLibraries = {
  libraries: [
    { key: "a1f0", name: "Films", folders: ["/data/films"] },
    { key: "b2e1", name: "Archive", folders: ["/data/archive"] },
  ],
  mappings: [
    {
      rootId: filmsRoot.id,
      rootName: "Films (/srv/media/films)",
      rootNode: homeRef,
      shown: true,
      serverFolder: "/data/films",
      library: { key: "a1f0", name: "Films", folder: "/data/films" },
      guess: false,
      confirmed: true,
      scanned: true,
    },
    {
      rootId: archiveRoot.id,
      rootName: "Archive (/mnt/archive/films)",
      rootNode: homeRef,
      shown: true,
      serverFolder: "/data/archive",
      library: { key: "b2e1", name: "Archive", folder: "/data/archive" },
      guess: true,
      confirmed: false,
      scanned: false,
      words:
        "Den Jellyfin won't get changes for files in Archive (/mnt/archive/films). Confirm the Path mapping, or mark the root as not shown on this server.",
    },
  ],
};

export const denChanges: MediaserversChangeView[] = [
  {
    id: "0199f3c1-3333-7000-8000-000000000004",
    kind: "added",
    path: "The Long Field (2019)/The Long Field (2019) WEBDL-1080p.mkv",
    state: "due",
    step: "none",
    lost: [],
    createdAt: "2026-11-14T09:25:00Z",
    attempts: 0,
    words: "This change is waiting to be sent.",
  },
  {
    id: "0199f3c1-3333-7000-8000-000000000003",
    kind: "removed",
    path: "Night Train (2018)/Night Train (2018) DVD.mkv",
    state: "due",
    step: "none",
    lost: [],
    createdAt: "2026-11-14T09:20:00Z",
    attempts: 2,
    words: "Den Jellyfin responded with status 500. SlipStream retries automatically.",
  },
  {
    id: "0199f3c1-3333-7000-8000-000000000002",
    kind: "moved",
    path: "The Quiet Orchard (2021)/The Quiet Orchard (2021) Bluray-1080p.mkv",
    fromPath: "The Quiet Orchard (2021)/The Quiet Orchard.mkv",
    state: "sent",
    step: "restored",
    result: "accepted",
    lost: ["play_count", "last_played"],
    createdAt: "2026-11-14T09:10:00Z",
    sentAt: "2026-11-14T09:13:00Z",
    attempts: 1,
    words: "Sent the rename. Lost: play count, last played.",
  },
  {
    id: "0199f3c1-3333-7000-8000-000000000001",
    kind: "added",
    path: "The Quiet Orchard (2021)/The Quiet Orchard.mkv",
    state: "sent",
    step: "none",
    result: "accepted",
    lost: [],
    createdAt: "2026-11-14T09:00:00Z",
    sentAt: "2026-11-14T09:00:10Z",
    attempts: 1,
    words: "Sent the new file.",
  },
];

export const denSessions: MediaserversSessionView[] = [
  {
    playId: "0199f3c2-4444-7000-8000-000000000001",
    user: "grace",
    player: "Living room TV",
    product: "Jellyfin for Android TV",
    path: "/data/films/The Quiet Orchard (2021)/The Quiet Orchard (2021) Bluray-1080p.mkv",
    workId: orchardId,
    decision: "transcode",
    hardware: true,
    reasons: ["ContainerNotSupported"],
    startedAt: "2026-11-14T20:00:00Z",
  },
];

export const denHolders: MediaserversHolderView[] = [
  { node: homeRef, fetchedAt: "2026-11-14T09:00:00Z", current: true },
];

export const plexAccount: MediaserversAccountView = {
  name: "ada",
  plexPass: true,
  signedInAt: "2026-11-10T09:00:00Z",
  expiresAt: "2026-11-17T09:00:00Z",
  renewedAt: "2026-11-14T03:00:00Z",
  refused: false,
};

export const plexServers: MediaserversPlexServerView[] = [
  {
    machineIdentifier: livingRoomPlex.identity,
    name: "Living room",
    version: livingRoomPlex.version,
    connections: [
      {
        uri: livingRoomPlex.address,
        local: true,
        https: true,
        reachedFrom: [
          { nodeId: home.id, ok: false },
          { nodeId: seedbox.id, ok: true },
        ],
      },
    ],
    suggestedRunsOn: seedboxRef,
    suggestedReaching: seedboxRef,
  },
];

const minute = 60;
const hour = 60 * minute;
const day = 24 * hour;

const duration = (
  key: string,
  label: string,
  value: number,
  [min, max, step]: readonly [number, number, number],
): SettingDurationField => ({
  kind: "duration",
  key,
  label,
  value,
  running: value,
  default: value,
  min,
  max,
  step,
  source: "default",
  applies: "live",
  pendingRestart: false,
  isDefault: true,
});

/** The Media Servers section's six settings at their defaults. */
export const mediaServerSettings = (): SettingDurationField[] => [
  duration("mediaServers.waitBeforeSending", "Wait before sending", 10, [0, 10 * minute, 1]),
  duration("mediaServers.sessionsEvery", "Check sessions every", 15, [5, 10 * minute, 1]),
  duration("mediaServers.unhealthyAfter", "Mark unhealthy after", 5 * minute, [
    minute,
    day,
    minute,
  ]),
  duration("mediaServers.keepPlays", "Keep play history for", 30 * day, [day, 365 * day, day]),
  duration("tasks.plex-sign-in.cadence", "Renew Plex sign-in every", day, [hour, 6 * day, hour]),
  duration("mediaServers.renewMargin", "Renew when less than this is left", 3 * day, [
    hour,
    6 * day,
    hour,
  ]),
];

const problem = (status: number, code: string, title: string, members: object = {}) =>
  HttpResponse.json(
    { status, code, title, ...members },
    { status, headers: { "Content-Type": "application/problem+json" } },
  );

const notFound = () => problem(404, "media_server_not_found", "No Media Server has this id");

/** Answers every Media Servers read from these servers, and Den Jellyfin's details for any server. */
export function mediaServerHandlers(list: MediaserversServerView[] = servers) {
  const byId = (id: string) => list.find((s) => s.id === id);
  return [
    http.get(`${api}/kinds`, () => HttpResponse.json({ kinds })),
    http.get(api, () => HttpResponse.json({ servers: list })),
    http.get(`${api}/settings`, () => HttpResponse.json({ settings: mediaServerSettings() })),
    http.get(`${api}/plex/account`, () => HttpResponse.json(plexAccount)),
    http.get(`${api}/plex/servers`, () => HttpResponse.json({ servers: plexServers })),
    http.get<{ id: string }>(`${api}/:id`, ({ params }) => {
      const s = byId(params.id);
      return s ? HttpResponse.json(s) : notFound();
    }),
    http.get<{ id: string }>(`${api}/:id/libraries`, ({ params }) =>
      byId(params.id) ? HttpResponse.json(denLibraries) : notFound(),
    ),
    http.get<{ id: string }>(`${api}/:id/changes`, ({ params }) =>
      byId(params.id) ? HttpResponse.json({ changes: denChanges }) : notFound(),
    ),
    http.get<{ id: string }>(`${api}/:id/sessions`, ({ params }) =>
      byId(params.id) ? HttpResponse.json({ reachable: true, sessions: denSessions }) : notFound(),
    ),
    http.get<{ id: string }>(`${api}/:id/credential-holders`, ({ params }) =>
      byId(params.id) ? HttpResponse.json({ holders: denHolders }) : notFound(),
    ),
  ];
}
