import { HttpResponse, http } from "msw";

import type {
  RootsRoot,
  SettingChoiceField,
  SettingField,
  SettingSwitchField,
  TargetsProfile,
  WorksCandidate,
  WorksContentType,
  WorksNewWork,
  WorksNewWorkPreset,
  WorksPresetCounts,
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
import { anthologies, extraCount, longShelf, storiesPerVolume, volumeCount } from "./anthology";
import { cinema, orchard } from "./library";
import { films, profileHandlers, ultra } from "./profiles";
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
  unlisted: false,
};

export const orchardHarvest: WorksCandidate = {
  key: "9000002",
  label: "Orchard Harvest",
  original: "",
  overview: "",
  fields: [{ key: "year", kind: "number", number: 2019 }],
  posterUrl: null,
  workId: harvestId,
  unlisted: false,
};

function previewOf(work: WorksWorkView, presets: WorksPresetCounts | null): WorksPreviewed {
  const {
    links: _links,
    profile: _profile,
    root: _root,
    monitorNew: _monitorNew,
    monitorAsides: _monitorAsides,
    levelFolders: _levelFolders,
    options: _options,
    ...tree
  } = work;
  return { inLibrary: false, work: tree, presets };
}

export const orchardPreview: WorksPreviewed = previewOf(orchard, null);

/** A second cinema root, on a volume the Node hasn't reported. */
export const shelfRoot: RootsRoot = {
  ...filmsRoot,
  id: "0199f0a1-1111-7000-8000-000000000003",
  path: "/mnt/shelf/films",
  default: false,
  volumeId: "sdb1",
};

const switchField = (key: string, label: string, value: boolean, base: boolean) =>
  ({
    key,
    label,
    kind: "switch",
    source: value === base ? "default" : "database",
    applies: "live",
    isDefault: value === base,
    pendingRestart: false,
    default: base,
    running: value,
    value,
  }) satisfies SettingSwitchField;

export const addMonitor = (value: boolean): SettingSwitchField =>
  switchField("cinema.add.monitor", "New Films are monitored", value, true);

/** The add defaults, the catalog and the Profiles and roots of one Content Type. */
export type Catalog = {
  readonly type: WorksContentType;
  readonly candidates: readonly WorksCandidate[];
  readonly preview: WorksPreviewed;
  readonly settings: readonly SettingField[];
  readonly profiles: readonly TargetsProfile[];
  readonly roots: readonly RootsRoot[];
};

export const cinemaCatalog = ({
  monitor = true,
  roots = [filmsRoot],
}: { monitor?: boolean; roots?: readonly RootsRoot[] } = {}): Catalog => ({
  type: cinema,
  candidates: [quietOrchard, orchardHarvest],
  preview: orchardPreview,
  settings: [addMonitor(monitor)],
  profiles: [films, ultra],
  roots,
});

export const longShelfCandidate: WorksCandidate = {
  key: longShelf.key,
  label: longShelf.label,
  original: "",
  overview: "Stories from a harbour town, one volume a year.",
  fields: [{ key: "publisher", kind: "text", text: "Harbor Press" }],
  posterUrl: null,
  workId: null,
  unlisted: false,
};

export const lostShelf: WorksCandidate = {
  key: "a2",
  label: "The Lost Shelf",
  original: "",
  overview: "Stories nobody catalogued.",
  fields: [{ key: "publisher", kind: "text", text: "Tide Books" }],
  posterUrl: null,
  workId: null,
  unlisted: true,
};

const mainStories = volumeCount * storiesPerVolume;

/** What each preset monitors of The Long Shelf, as the server counts it for a new Work. */
export const longShelfPresets: WorksPresetCounts = {
  asides: extraCount,
  choices: [
    { preset: "all", monitors: mainStories, monitorsNew: true },
    { preset: "future", monitors: 0, monitorsNew: true },
    { preset: "missing", monitors: mainStories, monitorsNew: true },
    { preset: "existing", monitors: 0, monitorsNew: false },
    { preset: "first", monitors: storiesPerVolume, monitorsNew: false },
    { preset: "latest", monitors: storiesPerVolume, monitorsNew: true },
    { preset: "none", monitors: 0, monitorsNew: false },
  ],
};

const presetLabels: { readonly [P in WorksNewWorkPreset]: string } = {
  all: "All",
  future: "Future",
  missing: "Missing",
  existing: "Existing",
  first: "First volume",
  latest: "Latest volume",
  none: "None",
};

export const anthologyMonitor = (value: WorksNewWorkPreset): SettingChoiceField => ({
  key: "anthology.add.monitor",
  label: "Stories to monitor",
  kind: "choice",
  source: value === "all" ? "default" : "database",
  applies: "live",
  isDefault: value === "all",
  pendingRestart: false,
  default: "all",
  running: value,
  value,
  options: Object.entries(presetLabels).map(([preset, label]) => ({ value: preset, label })),
});

export const shelves: TargetsProfile = {
  ...films,
  id: "0199e1f2-7a3b-7c4d-8e5f-00000000000a",
  name: "Shelves",
  contentType: anthologies.id,
};

export const anthologiesRoot: RootsRoot = {
  ...filmsRoot,
  id: "0199f0a1-1111-7000-8000-000000000004",
  contentType: anthologies.id,
  path: "/srv/media/anthologies",
};

export const anthologyCatalog = ({
  preset = "all",
  asides = false,
  levelFolders = true,
  candidates = [longShelfCandidate, lostShelf],
}: {
  preset?: WorksNewWorkPreset;
  asides?: boolean;
  levelFolders?: boolean;
  candidates?: readonly WorksCandidate[];
} = {}): Catalog => ({
  type: anthologies,
  candidates,
  preview: previewOf(longShelf, longShelfPresets),
  settings: [
    anthologyMonitor(preset),
    switchField("anthology.add.asides", "Include extras", asides, false),
    switchField("anthology.add.levelFolders", "Volume folders", levelFolders, true),
  ],
  profiles: [shelves],
  roots: [anthologiesRoot],
});

export type AddWorld = {
  readonly catalogs?: readonly Catalog[];
  readonly roots?: readonly RootsRoot[];
  readonly monitor?: boolean;
  readonly added?: WorksWorkView;
  readonly created?: boolean;
};

const catalogOf = (catalogs: readonly Catalog[], contentType: unknown): Catalog => {
  const catalog = catalogs.find(({ type }) => type.id === contentType);
  if (catalog === undefined) {
    throw new Error(`no catalog for ${String(contentType)}`);
  }
  return catalog;
};

/** Each Content Type's catalog, its preview and every read the add sheet makes, recording each search and add. */
export function addServer({
  roots,
  monitor,
  catalogs = [
    cinemaCatalog({
      ...(roots !== undefined && { roots }),
      ...(monitor !== undefined && { monitor }),
    }),
  ],
  added = orchard,
  created = true,
}: AddWorld = {}) {
  const searched: string[] = [];
  const posted: WorksNewWork[] = [];
  const handlers = [
    getListContentTypesMockHandler({ contentTypes: catalogs.map(({ type }) => type) }),
    getSearchCatalogMockHandler(({ request, params }) => {
      searched.push(new URL(request.url).searchParams.get("query") ?? "");
      return { candidates: [...catalogOf(catalogs, params["contentType"]).candidates] };
    }),
    getPreviewCatalogRecordMockHandler(
      ({ params }) => catalogOf(catalogs, params["contentType"]).preview,
    ),
    getGetContentTypeSettingsMockHandler(({ params }) => ({
      settings: [...catalogOf(catalogs, params["contentType"]).settings],
    })),
    getListRootsMockHandler({ roots: catalogs.flatMap((catalog) => catalog.roots) }),
    getListVolumesMockHandler({ nodes: [mediaVolumes] }),
    ...profileHandlers(catalogs.flatMap((catalog) => catalog.profiles)),
    http.post("*/api/v1/works", async ({ request }) => {
      posted.push(AddWorkBody.parse(await request.json()));
      return HttpResponse.json({ created, work: added });
    }),
    getGetWorkMockHandler(added),
  ];
  return { handlers, searched, posted };
}
