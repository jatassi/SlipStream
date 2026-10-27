import type {
  IndexersIndexer,
  IndexersQuery,
  IndexersTracker,
  NodesNode,
  SettingDurationField,
} from "../api/gen/model";
import { homeId, seedbox, seedboxId } from "./nodes";

const unlimited = { count: 0, per: "hour", used: 0, freeAt: null } as const;

const okStatus: IndexersIndexer["status"] = {
  state: "ok",
  cause: null,
  text: null,
  until: null,
  since: null,
  nodeId: null,
  solverTried: false,
  rung: 0,
  lastFailure: null,
  successAt: "2026-10-26T09:12:00Z",
};

export function indexer(over: Partial<IndexersIndexer> & Pick<IndexersIndexer, "id" | "name">) {
  return {
    kind: "torznab",
    homeNodeId: homeId,
    on: true,
    automaticSearch: true,
    rss: true,
    allowFailover: false,
    priority: 25,
    categories: null,
    media: null,
    allowNoSize: false,
    searches: unlimited,
    grabs: unlimited,
    status: okStatus,
    sentence: `${over.name} is working.`,
    fix: null,
    signsIn: false,
    trackerIds: [],
    waits: null,
    session: null,
    feedHost: "prowlarr.lan:9696",
    searchModes: [{ mode: "search", params: ["q"] }],
    revision: 3,
    createdAt: "2026-10-20T08:00:00Z",
    ...over,
  } satisfies IndexersIndexer;
}

export const torrentLeechId = "6b1e0c4a-2f3d-4e5b-8a7c-9d0e1f2a3b4c";
export const prowlarrId = "7c2f1d5b-3a4e-4f6c-9b8d-0e1f2a3b4c5d";
export const larkspurId = "8d3a2e6c-4b5f-4a7d-8c9e-1f2a3b4c5d6e";
export const torrentLeechTrackerId = "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d";
export const aitherTrackerId = "2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e";

export const torrentLeech = indexer({
  id: torrentLeechId,
  name: "TorrentLeech",
  kind: "catalog",
  homeNodeId: seedboxId,
  signsIn: true,
  feedHost: null,
  trackerIds: [torrentLeechTrackerId],
  searches: { count: 10, per: "hour", used: 4, freeAt: null },
  session: {
    nodeId: seedboxId,
    pasted: false,
    earnedAt: "2026-10-26T08:00:00Z",
  },
});

export const prowlarr = indexer({
  id: prowlarrId,
  name: "Prowlarr",
  trackerIds: [torrentLeechTrackerId, aitherTrackerId],
});

export const larkspur = indexer({
  id: larkspurId,
  name: "Larkspur",
  kind: "catalog",
  signsIn: true,
  feedHost: null,
  status: {
    ...okStatus,
    state: "stopped",
    cause: "credentials_rejected",
    since: "2026-10-26T07:00:00Z",
    nodeId: homeId,
  },
  waits: {
    kind: "stopped",
    releases: 0,
    until: null,
    since: null,
    nodeId: null,
    lapsed: false,
    cause: "credentials_rejected",
    text: null,
  },
  sentence: "Stopped because Larkspur rejected the username or password.",
  fix: "Update the username or password with Edit. Saving tests them.",
});

export const trackers: IndexersTracker[] = [
  {
    id: aitherTrackerId,
    name: "Aither",
    privacy: "private",
    hosts: [],
    keys: ["prowlarr:" + prowlarrId + ":3"],
    indexerIds: [prowlarrId],
  },
  {
    id: torrentLeechTrackerId,
    name: "TorrentLeech",
    privacy: "private",
    hosts: [],
    keys: ["definition:torrentleech"],
    indexerIds: [prowlarrId, torrentLeechId],
  },
];

/** seedbox holding Searching beside home, so an Indexer can move between them. */
export const searchingSeedbox: NodesNode = {
  ...seedbox,
  standing: "in-step",
  link: { state: "up", since: "2026-10-12T10:00:00Z" },
  roles: [{ role: "searching", state: "holding" }],
};

export const longBackoff: SettingDurationField = {
  source: "database",
  applies: "live",
  pendingRestart: false,
  isDefault: true,
  kind: "duration",
  key: "indexers.longBackoff",
  label: "Warn when every request has failed for",
  value: 21_600,
  running: 21_600,
  default: 21_600,
  min: 3600,
  max: 604_800,
  step: 3600,
};

export const queries: IndexersQuery[] = [
  {
    id: "q-3",
    use: "search",
    reservedAt: "2026-10-26T09:12:00Z",
    startedAt: "2026-10-26T09:12:00Z",
    completedAt: "2026-10-26T09:12:02Z",
    outcome: "ok",
    releases: 12,
  },
  {
    id: "q-2",
    use: "fetch",
    reservedAt: "2026-10-26T09:00:00Z",
    startedAt: "2026-10-26T09:00:00Z",
    completedAt: "2026-10-26T09:00:01Z",
    outcome: "failing",
    releases: 0,
  },
  {
    id: "q-1",
    use: "test",
    reservedAt: "2026-10-26T08:00:00Z",
    startedAt: "2026-10-26T08:00:00Z",
    completedAt: "2026-10-26T08:00:01Z",
    outcome: "ok",
    releases: 0,
  },
];
