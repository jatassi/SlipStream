import { HttpResponse, type JsonBodyType, http } from "msw";

import type {
  DownloadclientsClientCard,
  DownloadclientsClientsList,
  DownloadclientsSettings,
  DownloadclientsThresholds,
  SettingAddressField,
  SettingDurationField,
  SettingFolderField,
  SystemNodeConfig,
} from "../api/gen/model";
import { getListRootsMockHandler } from "../api/gen/query/roots/roots.msw";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { CheckFolderBody } from "../api/gen/zod/roots/roots.zod";
import { homeConfig } from "./node-system";
import { homeId, seedboxId } from "./nodes";
import { mediaVolume } from "./roots";

const about = {
  source: "database",
  applies: "live",
  pendingRestart: false,
  isDefault: true,
} as const;

const duration = (
  key: string,
  label: string,
  value: number,
  [min, max, step]: readonly [number, number, number],
): SettingDurationField => ({
  ...about,
  kind: "duration",
  key,
  label,
  value,
  running: value,
  default: value,
  min,
  max,
  step,
});

export const thresholds: DownloadclientsThresholds = {
  faultStall: duration(
    "downloading.faultStall",
    "A download with no data at all is bad after",
    43_200,
    [3600, 2_592_000, 3600],
  ),
  unregisteredGap: duration(
    "downloading.unregisteredGap",
    "Unregistered twice, this far apart",
    1800,
    [60, 604_800, 60],
  ),
};

const mib = 1024 * 1024;

export const clientSettings: DownloadclientsSettings = {
  downloadLimit: {
    ...about,
    kind: "rate",
    key: "downloading.downloadLimit",
    label: "Download limit",
    value: 0,
    running: 0,
    default: 0,
    max: 2000 * mib,
    zero: "No limit",
  },
  uploadLimit: {
    ...about,
    kind: "rate",
    key: "downloading.uploadLimit",
    label: "Upload limit",
    value: 0,
    running: 0,
    default: 0,
    max: 2000 * mib,
    zero: "No limit",
  },
  activeDownloads: {
    ...about,
    kind: "count",
    key: "downloading.activeDownloads",
    label: "Downloads at once",
    value: 3,
    running: 3,
    default: 3,
    min: 0,
    max: 1000,
    unit: { one: "download", many: "downloads" },
    zero: "No limit",
  },
  activeChecks: {
    ...about,
    kind: "count",
    key: "downloading.activeChecks",
    label: "File checks at once",
    value: 1,
    running: 1,
    default: 1,
    min: 1,
    max: 50,
    unit: { one: "check", many: "checks" },
  },
  connections: {
    ...about,
    kind: "count",
    key: "downloading.connections",
    label: "Peer connections",
    value: 8000,
    running: 8000,
    default: 8000,
    min: 50,
    max: 65_535,
    unit: { one: "connection", many: "connections" },
  },
  utp: {
    ...about,
    kind: "switch",
    key: "downloading.utp",
    label: "uTP, incoming and outgoing",
    value: true,
    running: true,
    default: true,
  },
  stalledAfter: duration("downloading.stalledAfter", "Stalled after", 600, [60, 86_400, 60]),
};

const limits = {
  downloadLimit: 0,
  uploadLimit: 4 * mib,
  activeDownloads: 3,
  activeChecks: 1,
  connections: 8000,
  utp: true,
  stalledAfterSeconds: 600,
};

/** A built-in client's facts before it has a listener. */
export const unbound = {
  libtorrent: "2.0.11",
  peerIdPrefix: "-SS0400-",
  userAgent: "SlipStream/0.4.0 libtorrent/2.0.11",
  listen: "0.0.0.0:6881",
  reach: "unproven",
} as const;

export const homeCard: DownloadclientsClientCard = {
  nodeId: homeId,
  nodeName: "home",
  inStep: true,
  client: { ...unbound, bound: "192.168.1.20:6881" },
  limits,
  torrents: 12,
  totals: {
    uploaded: 3 * 1024 ** 4,
    downloaded: 900 * 1024 ** 3,
    seedingSeconds: 6 * 86_400,
    activeSeconds: 2 * 86_400,
  },
};

export const seedboxCard: DownloadclientsClientCard = {
  nodeId: seedboxId,
  nodeName: "seedbox",
  inStep: true,
  client: {
    ...unbound,
    listen: "203.0.113.9:51413",
    bound: "203.0.113.9:51413",
    reach: "reachable",
    reachedAt: "2026-10-12T09:05:00Z",
  },
  limits,
  torrents: 1,
  totals: { uploaded: 0, downloaded: 0, seedingSeconds: 0, activeSeconds: 0 },
};

export const listenSetting: SettingAddressField = {
  ...about,
  source: "default",
  applies: "restart",
  key: "torrent.listen",
  label: "Listen address",
  kind: "address",
  value: "0.0.0.0:6881",
  running: "0.0.0.0:6881",
  default: "0.0.0.0:6881",
};

export const downloadsSetting: SettingFolderField = {
  ...about,
  source: "default",
  applies: "restart",
  key: "torrent.downloads",
  label: "Downloads folder",
  kind: "folder",
  value: "/data/home/downloads",
  running: "/data/home/downloads",
  default: "/data/home/downloads",
  nodeId: homeId,
  purpose: "downloads",
};

/** home's config with the torrent section, its rows replaced by `rows`. */
export const withTorrent = (
  rows: { listen?: Partial<SettingAddressField>; downloads?: Partial<SettingFolderField> } = {},
): SystemNodeConfig => ({
  sections: [
    ...homeConfig.sections,
    {
      id: "torrent",
      fields: [
        { ...listenSetting, ...rows.listen },
        { ...downloadsSetting, ...rows.downloads },
      ],
    },
  ],
});

export type ClientWrite = { path: string; body: JsonBodyType };

/**
 * Serves the Download Clients list and each Node's client settings, and
 * records every PATCH. A test changes `state` to change what is served.
 */
export function downloadClients(list?: DownloadclientsClientsList) {
  const state = { list: list ?? { cards: [homeCard], thresholds }, settings: clientSettings };
  const writes: ClientWrite[] = [];
  const handlers = [
    http.get("*/api/v1/download-clients", () => HttpResponse.json(state.list)),
    http.patch("*/api/v1/download-clients/thresholds", async ({ request }) => {
      writes.push({ path: "thresholds", body: await request.json() });
      return HttpResponse.json(state.list.thresholds);
    }),
    http.get("*/api/v1/download-clients/:nodeId/settings", () => HttpResponse.json(state.settings)),
    http.patch("*/api/v1/download-clients/:nodeId/settings", async ({ request, params }) => {
      writes.push({ path: `${String(params["nodeId"])}/settings`, body: await request.json() });
      return HttpResponse.json(state.settings);
    }),
  ];
  return { handlers, state, writes };
}

/** No roots and no Content Types, and every downloads folder usable: what the Downloads folder field reads. */
export const usableFolders = [
  getListRootsMockHandler({ roots: [] }),
  getListContentTypesMockHandler({ contentTypes: [] }),
  http.post("*/api/v1/nodes/:id/folders/check", async ({ request }) => {
    const { path } = CheckFolderBody.parse(await request.json());
    return HttpResponse.json({ usable: true, path, volume: mediaVolume });
  }),
];
