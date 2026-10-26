import { HttpResponse, http } from "msw";

import type {
  RootsRoot,
  SettingSwitchField,
  WorksCandidate,
  WorksNewWork,
  WorksPreviewed,
  WorksWorkView,
} from "../api/gen/model";
import {
  getListRootsMockHandler,
  getListVolumesMockHandler,
} from "../api/gen/query/roots/roots.msw";
import {
  getGetContentTypeSettingsMockHandler,
  getGetWorkMockHandler,
  getListContentTypesMockHandler,
  getPreviewCatalogRecordMockHandler,
  getSearchCatalogMockHandler,
} from "../api/gen/query/works/works.msw";
import { AddWorkBody } from "../api/gen/zod/works/works.zod";
import { cinema, orchard } from "./library";
import { profileHandlers } from "./profiles";
import { filmsRoot, mediaVolumes } from "./roots";

export const harvestId = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";

export const quietOrchard: WorksCandidate = {
  key: "9000001",
  label: "The Quiet Orchard",
  original: "Le Verger Silencieux",
  overview: "An orchard keeper waits out a long winter.",
  fields: [{ key: "year", kind: "number", number: 2021 }],
  posterUrl: null,
  workId: null,
};

export const orchardHarvest: WorksCandidate = {
  key: "9000002",
  label: "Orchard Harvest",
  original: "",
  overview: "",
  fields: [{ key: "year", kind: "number", number: 2019 }],
  posterUrl: null,
  workId: harvestId,
};

const { links: _links, profile: _profile, root: _root, ...tree } = orchard;

export const orchardPreview: WorksPreviewed = { inLibrary: false, work: tree };

/** A second cinema root, on a volume the Node hasn't reported. */
export const shelfRoot: RootsRoot = {
  ...filmsRoot,
  id: "0199f0a1-1111-7000-8000-000000000003",
  path: "/mnt/shelf/films",
  default: false,
  volumeId: "sdb1",
};

export const addMonitor = (value: boolean): SettingSwitchField => ({
  key: "cinema.add.monitor",
  label: "New Films are monitored",
  kind: "switch",
  source: value ? "default" : "database",
  applies: "live",
  isDefault: value,
  pendingRestart: false,
  default: true,
  running: value,
  value,
});

export type AddWorld = {
  readonly roots?: readonly RootsRoot[];
  readonly monitor?: boolean;
  readonly added?: WorksWorkView;
  readonly created?: boolean;
};

/** The catalog, its preview and every read the add sheet makes, recording each search and add. */
export function addServer({
  roots = [filmsRoot],
  monitor = true,
  added = orchard,
  created = true,
}: AddWorld = {}) {
  const searched: string[] = [];
  const posted: WorksNewWork[] = [];
  const handlers = [
    getListContentTypesMockHandler({ contentTypes: [cinema] }),
    getSearchCatalogMockHandler(({ request }) => {
      searched.push(new URL(request.url).searchParams.get("query") ?? "");
      return { candidates: [quietOrchard, orchardHarvest] };
    }),
    getPreviewCatalogRecordMockHandler(orchardPreview),
    getGetContentTypeSettingsMockHandler({ settings: [addMonitor(monitor)] }),
    getListRootsMockHandler({ roots: [...roots] }),
    getListVolumesMockHandler({ nodes: [mediaVolumes] }),
    ...profileHandlers(),
    http.post("*/api/v1/works", async ({ request }) => {
      posted.push(AddWorkBody.parse(await request.json()));
      return HttpResponse.json({ created, work: added });
    }),
    getGetWorkMockHandler(added),
  ];
  return { handlers, searched, posted };
}
