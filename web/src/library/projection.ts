import type { QueryClient } from "@tanstack/react-query";

import type { WorksContentType, WorksLibraryRow, WorksLibraryRows } from "../api/gen/model";
import {
  getListLibrarySuspenseQueryOptions,
  useListContentTypesSuspense,
  useListLibrarySuspense,
} from "../api/gen/query/works/works";
import { ensureQuery } from "../api/query-client";
import { typeOf } from "../works/fields";
import { type LibraryWork, parseRow } from "./rows";

/** Every Work of every Content Type, parsed, with its two lookups. */
export type Projection = {
  readonly works: readonly LibraryWork[];
  readonly byId: ReadonlyMap<string, LibraryWork>;
  /** By Content Type and the Work's key in its catalog, for "In the library". */
  readonly byKey: ReadonlyMap<string, LibraryWork>;
};

const keyOf = (type: string, key: string) => `${type}/${key}`;

/** Loads the one cached read of the whole Library, which only live events move. */
export const ensureLibrary = (queryClient: QueryClient): Promise<WorksLibraryRows> =>
  ensureQuery(queryClient, getListLibrarySuspenseQueryOptions());

const projections = new WeakMap<
  readonly WorksLibraryRow[],
  { readonly types: readonly WorksContentType[]; readonly projection: Projection }
>();

function projectionOf(rows: readonly WorksLibraryRow[], types: WorksContentType[]): Projection {
  const known = projections.get(rows);
  if (known?.types === types) {
    return known.projection;
  }
  const works = rows.map((row) => parseRow(row, typeOf(types, row.contentType)));
  const projection: Projection = {
    works,
    byId: new Map(works.map((work) => [work.id, work])),
    byKey: new Map(works.map((work) => [keyOf(work.type.id, work.key), work])),
  };
  projections.set(rows, { types, projection });
  return projection;
}

export function useProjection(): Projection {
  const { rows } = useListLibrarySuspense().data;
  const types = useListContentTypesSuspense().data.contentTypes;
  return projectionOf(rows, types);
}

/** Every Work in the Library, in the server's order: sort label, then id. */
export const useLibrary = (): readonly LibraryWork[] => useProjection().works;

export const useLibraryWork = (id: string): LibraryWork | undefined => useProjection().byId.get(id);

/** Finds the Work a catalog result is, when it is in the Library. */
export function useWorkByKey(): (type: string, key: string) => LibraryWork | undefined {
  const { byKey } = useProjection();
  return (type, key) => byKey.get(keyOf(type, key));
}
