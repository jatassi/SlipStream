import type {
  WorksContentType,
  WorksLibraryRow,
  WorksListedWork,
  WorksWorkView,
} from "../api/gen/model";
import { getListLibraryMockHandler } from "../api/gen/query/works/works.msw";

export const cinema: WorksContentType = {
  id: "cinema",
  name: "Films",
  icon: "film",
  enabled: true,
  work: { one: "Film", many: "Films" },
  item: { one: "Film", many: "Films" },
  levels: [],
  fields: [
    {
      key: "year",
      label: "Year",
      tier: "work",
      kind: "number",
      in: ["facts"],
      list: { column: true, sort: true, filter: true },
      with: null,
      art: null,
    },
    {
      key: "released",
      label: "Released",
      tier: "work",
      kind: "date",
      in: ["about"],
      list: { column: false, sort: false, filter: false },
      with: null,
      art: null,
    },
    {
      key: "runtime",
      label: "Runtime",
      tier: "work",
      kind: "duration",
      in: ["facts"],
      list: { column: true, sort: true, filter: false },
      with: null,
      art: null,
    },
  ],
  artwork: [],
  source: "TMDB",
  items: "one",
  art: { poster: null, backdrop: null, title: null },
  ratings: [],
};

export const orchardId = "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03";
export const orchardItemId = "8e2b4d6f-1a3c-4e5f-9b7d-0c2e4a6b8d1f";

export const orchard: WorksWorkView = {
  id: orchardId,
  contentType: "cinema",
  key: "9000001",
  label: "The Quiet Orchard",
  sortLabel: "Quiet Orchard",
  addedAt: "2026-10-12T09:00:00Z",
  refreshedAt: "2026-10-12T09:00:00Z",
  goneUpstreamAt: null,
  fields: [
    { key: "year", kind: "number", number: 2021 },
    { key: "released", kind: "date", date: "2021-04-09" },
    { key: "runtime", kind: "duration", ms: 6_720_000 },
  ],
  ratings: [],
  artwork: [],
  levels: [],
  items: [
    {
      id: orchardItemId,
      levelId: null,
      key: "film",
      label: "The Quiet Orchard",
      position: 1,
      withdrawnAt: null,
      monitored: true,
      fields: [],
      ratings: [],
      artwork: [],
      availability: { date: "2021-04-09", basis: "Digital release in United States" },
    },
  ],
  links: [],
  profile: { id: "0199e1f2-7a3b-7c4d-8e5f-000000000001", name: "Films" },
  root: null,
};

export const orchardListed: WorksListedWork = {
  id: orchard.id,
  contentType: orchard.contentType,
  key: orchard.key,
  label: orchard.label,
  sortLabel: orchard.sortLabel,
  addedAt: orchard.addedAt,
  refreshedAt: orchard.refreshedAt,
  goneUpstreamAt: orchard.goneUpstreamAt,
  fields: orchard.fields,
  artwork: orchard.artwork,
};

export const orchardRow: WorksLibraryRow = {
  id: orchardId,
  contentType: "cinema",
  key: orchard.key,
  label: orchard.label,
  sortLabel: orchard.sortLabel,
  addedAt: orchard.addedAt,
  refreshedAt: orchard.refreshedAt,
  goneUpstreamAt: null,
  monitored: "all",
  profileId: orchard.profile.id,
  rootId: null,
  posterUrl: null,
  rating: null,
  availableOn: "2021-04-09",
  bytes: 0,
  files: 0,
  kept: 0,
  status: "missing",
  cells: { missing: 2 },
  failed: 0,
  items: 1,
  itemStatus: { missing: 1 },
  needsYou: 0,
  values: { year: 2021, runtime: 6_720_000 },
};

/** A cinema row like the Quiet Orchard's, with its own id, key and label. */
export const libraryRow = (
  row: Partial<WorksLibraryRow> & Pick<WorksLibraryRow, "id" | "key" | "label">,
): WorksLibraryRow => ({ ...orchardRow, sortLabel: row.label, ...row });

/**
 * GET /library over `rows`, which `become` replaces: every row, or the
 * named ones. `asked` records the ids of each named read.
 */
export function libraryServer(initial: readonly WorksLibraryRow[]) {
  let rows = [...initial];
  const asked: string[][] = [];
  return {
    handler: getListLibraryMockHandler(({ request }) => {
      const ids = new URL(request.url).searchParams.get("workIds");
      if (ids === null) {
        return { rows };
      }
      const named = ids.split(",");
      asked.push(named);
      return { rows: rows.filter((row) => named.includes(row.id)) };
    }),
    asked,
    become(next: readonly WorksLibraryRow[]) {
      rows = [...next];
    },
  };
}
