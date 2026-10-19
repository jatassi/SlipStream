import { HttpResponse, http } from "msw";

import type {
  CheckRootInputBody,
  NodesNode,
  RehomeRootInputBody,
  RootsNewRoot,
  RootsNodeVolumes,
  RootsRoot,
  RootsVerdict,
  RootsVolume,
  SettingDurationField,
} from "../api/gen/model";
import {
  CheckRootBody,
  CreateRootBody,
  RehomeRootBody,
  UpdateRootSettingsBody,
} from "../api/gen/zod/roots/roots.zod";
import type { liveServer } from "./live-server";
import { home, seedbox } from "./nodes";
import { worker } from "./worker";

/** The Controller, holding Storage as well as its other Roles. */
export const media: NodesNode = {
  ...home,
  roles: [...home.roles, { role: "storage", state: "holding" }],
};

/** An offline Worker that doesn't hold Storage. */
export const notStorage: NodesNode = seedbox;

export const filmsRoot: RootsRoot = {
  id: "0199f0a1-1111-7000-8000-000000000001",
  contentType: "cinema",
  nodeId: media.id,
  path: "/srv/media/films",
  default: true,
  state: "ready",
  volumeId: "sda1",
  works: 12,
  fileBytes: 910 * 1024 ** 3,
};

export const archiveRoot: RootsRoot = {
  id: "0199f0a1-1111-7000-8000-000000000002",
  contentType: "cinema",
  nodeId: media.id,
  path: "/mnt/archive/films",
  default: false,
  state: "unmounted",
  works: 3,
  fileBytes: 0,
};

export const mediaVolume: RootsVolume = {
  id: "sda1",
  name: "/srv/media",
  mount: "/srv/media",
  kind: "local",
  total: 8 * 1024 ** 4,
  free: 1.2 * 1024 ** 4,
  sampledAt: "2026-10-19T14:05:00Z",
};

export const mediaVolumes: RootsNodeVolumes = {
  nodeId: media.id,
  live: true,
  volumes: [{ ...mediaVolume, rootIds: [filmsRoot.id] }],
};

export const sampleEvery: SettingDurationField = {
  key: "storage.sampleEvery",
  label: "Check storage every",
  kind: "duration",
  source: "default",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
  value: 300,
  running: 300,
  default: 300,
  min: 60,
  max: 86_400,
  step: 60,
};

const problemType = { "Content-Type": "application/problem+json" };

export const refused = (status: number, code: string, title: string, members: object = {}) =>
  HttpResponse.json({ status, code, title, ...members }, { status, headers: problemType });

type Answer = Response | Promise<Response>;

export type RootsWorld = {
  roots: RootsRoot[];
  volumes: RootsNodeVolumes[];
  sampleEvery: SettingDurationField;
  /** The verdict on a folder; usable on its volume unless a test says otherwise. */
  check: (body: CheckRootInputBody) => RootsVerdict;
  rehome: (rootId: string, body: RehomeRootInputBody) => Answer;
  remove: (rootId: string, worksTo: string | null) => Answer;
  /** Publishes roots.changed, as the Controller does after each change. */
  changed: (rootIds: string[]) => void;
};

const usableOnMedia = ({ path }: CheckRootInputBody): RootsVerdict => ({
  usable: true,
  path,
  volume: mediaVolume,
});

type Sent = {
  created: RootsNewRoot[];
  checked: CheckRootInputBody[];
  rehomed: RehomeRootInputBody[];
  defaulted: string[];
  deleted: { rootId: string; worksTo: string | null }[];
  sampleEvery: (number | null | undefined)[];
};

function madeFrom(world: RootsWorld, body: RootsNewRoot, count: number): RootsRoot {
  return {
    id: `0199f0a1-2222-7000-8000-${String(count).padStart(12, "0")}`,
    contentType: body.contentType,
    nodeId: body.nodeId,
    path: body.path,
    default:
      body.makeDefault === true ||
      !world.roots.some((root) => root.contentType === body.contentType),
    state: "checking",
    works: 0,
    fileBytes: 0,
  };
}

const readsOf = (world: RootsWorld, sent: Sent) => [
  http.get("*/api/v1/roots", () => HttpResponse.json({ roots: world.roots })),
  http.get("*/api/v1/volumes", () => HttpResponse.json({ nodes: world.volumes })),
  http.get("*/api/v1/roots/settings", () => HttpResponse.json({ sampleEvery: world.sampleEvery })),
  http.patch("*/api/v1/roots/settings", async ({ request }) => {
    const { sampleEvery: value } = UpdateRootSettingsBody.parse(await request.json());
    sent.sampleEvery.push(value);
    world.sampleEvery = {
      ...world.sampleEvery,
      value: value ?? world.sampleEvery.default,
      isDefault: value === null || value === undefined,
    };
    return HttpResponse.json({ sampleEvery: world.sampleEvery });
  }),
  http.post("*/api/v1/roots/check", async ({ request }) => {
    const body = CheckRootBody.parse(await request.json());
    sent.checked.push(body);
    return HttpResponse.json(world.check(body));
  }),
];

const writesOf = (world: RootsWorld, sent: Sent) => [
  http.post("*/api/v1/roots", async ({ request }) => {
    const body = CreateRootBody.parse(await request.json());
    sent.created.push(body);
    const made = madeFrom(world, body, sent.created.length);
    world.roots = [...world.roots, made];
    world.changed([made.id]);
    return HttpResponse.json(made, { status: 201 });
  }),
  http.post("*/api/v1/roots/:rootId/default", ({ params }) => {
    sent.defaulted.push(String(params["rootId"]));
    return new HttpResponse(null, { status: 204 });
  }),
  http.patch("*/api/v1/roots/:rootId", async ({ params, request }) => {
    const body = RehomeRootBody.parse(await request.json());
    sent.rehomed.push(body);
    return world.rehome(String(params["rootId"]), body);
  }),
  http.delete("*/api/v1/roots/:rootId", ({ params, request }) => {
    const rootId = String(params["rootId"]);
    const worksTo = new URL(request.url).searchParams.get("worksTo");
    sent.deleted.push({ rootId, worksTo });
    return world.remove(rootId, worksTo);
  }),
];

/** Serves the roots endpoints from one mutable world, recording what the page sent. */
export function serveRoots(start: Partial<RootsWorld> = {}) {
  const world: RootsWorld = {
    roots: [filmsRoot],
    volumes: [mediaVolumes],
    sampleEvery,
    check: usableOnMedia,
    rehome: () => refused(500, "unexpected", "No re-home expected"),
    remove: (rootId) => {
      world.roots = world.roots.filter((root) => root.id !== rootId);
      world.changed([rootId]);
      return new HttpResponse(null, { status: 204 });
    },
    changed: () => {},
    ...start,
  };
  const sent: Sent = {
    created: [],
    checked: [],
    rehomed: [],
    defaulted: [],
    deleted: [],
    sampleEvery: [],
  };
  worker.use(...readsOf(world, sent), ...writesOf(world, sent));
  return { world, sent };
}

/** Sends roots.changed to the page over a test's live stream. */
export const changedOver =
  (live: ReturnType<typeof liveServer>) =>
  (rootIds: string[]): void => {
    live.clients[0]?.send({ event: "roots.changed", data: { rootIds } });
  };
