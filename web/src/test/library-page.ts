import type { WorksLibraryRow } from "../api/gen/model";
import {
  getListRootsMockHandler,
  getListVolumesMockHandler,
} from "../api/gen/query/roots/roots.msw";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
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
