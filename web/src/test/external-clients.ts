import { HttpResponse, type JsonBodyType, http } from "msw";

import type {
  DownloadclientsClientCard,
  DownloadclientsExternalClient,
  DownloadclientsKind,
  DownloadclientsTestReport,
  SettingDurationField,
} from "../api/gen/model";
import {
  CreateDownloadClientBody,
  UpdateDownloadClientBody,
} from "../api/gen/zod/downloadclients/downloadclients.zod";
import { homeId, seedboxId } from "./nodes";

const timing = (
  key: string,
  label: string,
  value: number,
  [min, max, step]: readonly [number, number, number],
): SettingDurationField => ({
  source: "database",
  applies: "live",
  pendingRestart: false,
  isDefault: true,
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

const acts = ["pause", "resume", "recheck", "announce"] as const;

export const qbittorrentKind: DownloadclientsKind = {
  kind: "qbittorrent",
  name: "qBittorrent",
  path: "",
  username: "optional",
  password: "optional",
  apiKey: true,
  daemon: false,
  label: "any",
  floor: "4.6",
  actions: [...acts],
  deletesData: true,
};

export const transmissionKind: DownloadclientsKind = {
  kind: "transmission",
  name: "Transmission",
  path: "/transmission/rpc",
  username: "optional",
  password: "optional",
  apiKey: false,
  daemon: false,
  label: "no_comma",
  floor: "4.0",
  actions: [...acts],
  deletesData: true,
};

export const delugeKind: DownloadclientsKind = {
  kind: "deluge",
  name: "Deluge",
  path: "/json",
  username: "none",
  password: "required",
  apiKey: false,
  daemon: true,
  label: "lower_word",
  floor: "2.1",
  actions: [...acts],
  deletesData: true,
};

export const rtorrentKind: DownloadclientsKind = {
  kind: "rtorrent",
  name: "rTorrent",
  path: "/RPC2",
  username: "optional",
  password: "optional",
  apiKey: false,
  daemon: false,
  label: "any",
  floor: "0.9.8",
  actions: [...acts],
  deletesData: false,
};

export const kinds = [qbittorrentKind, transmissionKind, delugeKind, rtorrentKind];

/** Transmission on home, responding at 4.1.0 and holding 3 torrents. */
export const attic: DownloadclientsExternalClient = {
  id: "7d3f9b1e-5a2c-4e8f-b6d0-1c3e5a7f9b2d",
  nodeId: homeId,
  kind: "transmission",
  name: "Transmission",
  address: "http://192.168.1.20:9091/transmission/rpc",
  webUi: "http://192.168.1.20:9091/transmission/web/",
  username: "jackson",
  passwordSaved: true,
  apiKeySaved: false,
  label: "slipstream",
  folder: "",
  clientFolders: [],
  daemon: null,
  checkEvery: timing("downloadClients.checkEvery", "Check every", 10, [2, 3600, 1]),
  importDelay: timing("downloadClients.importDelay", "Import delay", 0, [0, 86_400, 60]),
  on: true,
  settingsVersion: 1,
  reach: "responding",
  away: false,
  since: "2026-11-07T09:00:00Z",
  productVersion: "4.1.0",
  problem: null,
  torrents: 3,
  actions: [...acts],
};

export const atticAway: DownloadclientsExternalClient = {
  ...attic,
  reach: "not_responding",
  away: true,
  since: "2026-11-07T14:02:00Z",
  problem: {
    cause: "no_response",
    sentence:
      "Transmission isn't responding. Check that it's running and that home can reach 192.168.1.20:9091.",
    fields: ["address"],
  },
};

export const atticRefused: DownloadclientsExternalClient = {
  ...attic,
  reach: "sign_in_refused",
  away: true,
  problem: {
    cause: "wrong_credentials",
    sentence:
      "Transmission rejected the saved password. SlipStream stopped checking it to avoid a lockout. Change the password, or select Check now once it's fixed.",
    fields: ["username", "password"],
  },
};

/** qBittorrent on seedbox, a Node that no longer downloads. */
export const orphan: DownloadclientsExternalClient = {
  ...attic,
  id: "2b4d6f8a-0c1e-4a3b-9d5f-7e9a1c3b5d7f",
  nodeId: seedboxId,
  kind: "qbittorrent",
  name: "qBittorrent",
  address: "http://203.0.113.9:8080",
  webUi: "http://203.0.113.9:8080/",
  username: "",
  passwordSaved: false,
  apiKeySaved: true,
  reach: "unchecked",
  since: null,
  productVersion: "",
  torrents: 0,
};

export const withExternals = (
  card: DownloadclientsClientCard,
  ...externals: DownloadclientsExternalClient[]
): DownloadclientsClientCard => ({ ...card, externals });

export const passed: DownloadclientsTestReport = {
  product: "Transmission",
  version: "4.1.1",
  nodeFolder: "/downloads",
  warnings: [],
  daemons: [],
};

/** A problem response carrying more than the Problem's own members, such as a failed Test's fields. */
export const refusal = (status: number, body: Record<string, JsonBodyType>) =>
  HttpResponse.json(
    { status, ...body },
    { status, headers: { "Content-Type": "application/problem+json" } },
  );

/** How the external clients' operations answer: a create and an update with the name they sent. */
export type Answers = {
  test: () => Response;
  create: (name: string | undefined) => Response;
  update: (name: string | undefined) => Response;
  remove: () => Response;
};

export const passing = (): Answers => ({
  test: () => HttpResponse.json(passed),
  create: (name) => HttpResponse.json({ ...attic, name: name ?? attic.name }),
  update: (name) => HttpResponse.json({ ...attic, name: name ?? attic.name }),
  remove: () => new HttpResponse(null, { status: 204 }),
});

export type Recorder = (request: Request, path: string) => Promise<unknown>;

const external = "*/api/v1/external-download-clients";

/** The external clients' operations, each answered by its entry in answers. */
export const externalHandlers = (answers: () => Answers, record: Recorder) => [
  http.post(`${external}/test`, async ({ request }) => {
    await record(request, "test");
    return answers().test();
  }),
  http.post(`${external}/:id/test`, async ({ request, params }) => {
    await record(request, `${String(params["id"])}/test`);
    return answers().test();
  }),
  http.post(external, async ({ request }) =>
    answers().create(CreateDownloadClientBody.parse(await record(request, "new")).name),
  ),
  http.patch(`${external}/:id`, async ({ request, params }) =>
    answers().update(
      UpdateDownloadClientBody.parse(await record(request, String(params["id"]))).name,
    ),
  ),
  http.delete(`${external}/:id`, async ({ request, params }) => {
    await record(request, String(params["id"]));
    return answers().remove();
  }),
];
