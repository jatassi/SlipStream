import type { WorksLibraryRow } from "../api/gen/model";
import {
  getListRootsMockHandler,
  getListVolumesMockHandler,
} from "../api/gen/query/roots/roots.msw";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { anthologies } from "./anthology";
import { cinema, libraryRow, libraryServer, orchardRow } from "./library";
import { profileHandlers, ultra } from "./profiles";
import { filmsRoot, mediaVolumes } from "./roots";

/** Seen in 2019, met and in the films root. */
export const harvest = libraryRow({
  id: "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b",
  key: "9000002",
  label: "Orchard Harvest",
  addedAt: "2026-10-20T09:00:00Z",
  availableOn: "2019-05-01",
  status: "met",
  itemStatus: { met: 1 },
  bytes: 4 * 1024 ** 3,
  files: 1,
  rootId: filmsRoot.id,
  profileId: ultra.id,
  values: { year: 2019, runtime: 5_400_000 },
});

/** Unmonitored, with a failed search. */
export const ember = libraryRow({
  id: "6e8a0c2d-4f6b-4d8f-8a2c-4e6a8c0e2f4a",
  key: "9000003",
  label: "Ember Lane",
  addedAt: "2025-01-05T09:00:00Z",
  availableOn: "2024-03-01",
  status: "failed",
  failed: 1,
  monitored: "none",
  values: { year: 2024, runtime: 7_800_000 },
});

export const threeFilms: readonly WorksLibraryRow[] = [orchardRow, harvest, ember];

/** A cinema Library of `rows`, with its Profiles, its root and the root's volume. */
export function cinemaLibrary(rows: readonly WorksLibraryRow[] = threeFilms) {
  const library = libraryServer(rows);
  return {
    library,
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      library.handler,
      ...profileHandlers(),
      getListRootsMockHandler({ roots: [filmsRoot] }),
      getListVolumesMockHandler({ nodes: [mediaVolumes] }),
    ],
  };
}

/** `count` cinema rows labelled Orchard 1, Orchard 2 and on. */
export const orchards = (count: number): WorksLibraryRow[] =>
  Array.from({ length: count }, (_, index) =>
    libraryRow({
      id: `${orchardRow.id.slice(0, -2)}${String(index).padStart(2, "0")}`,
      key: String(9_100_000 + index),
      label: `Orchard ${index + 1}`,
    }),
  );

/** An anthology with 12 of its 40 counted Stories met, the next issue due in December. */
export const harborTales = libraryRow({
  id: "7f9b1d3e-5a7c-4e9a-8b3d-5f7a9c1e3b5d",
  contentType: "anthology",
  key: "anthology-1",
  label: "Harbor Tales",
  availableOn: "2026-12-01",
  status: "missing",
  cells: { met: 12, missing: 28 },
  items: 40,
  itemStatus: { met: 12, missing: 28 },
  values: { publisher: "Harbor Press" },
});

/** A Library of films and anthologies, both types enabled. */
export function twoTypeLibrary(rows: readonly WorksLibraryRow[] = [...threeFilms, harborTales]) {
  return [
    getListContentTypesMockHandler({ contentTypes: [cinema, anthologies] }),
    libraryServer(rows).handler,
    ...profileHandlers(),
    getListRootsMockHandler({ roots: [filmsRoot] }),
    getListVolumesMockHandler({ nodes: [mediaVolumes] }),
  ];
}
