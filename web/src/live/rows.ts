import type { WorksLibraryRows } from "../api/gen/model";
import { getListLibraryQueryKey, listLibrary } from "../api/gen/query/works/works";
import { createSplicer } from "./splice";

const library = createSplicer<WorksLibraryRows>({
  queryKey: getListLibraryQueryKey(),
  read: (ids) => listLibrary({ workIds: [...ids] }),
});

/** Reads the named Works' Library rows again, coalesced per frame. */
export const queueRows = library.queue;

/** Drops the named Works' Library rows. */
export const dropRows = library.drop;

/** The whole Library, for a change too wide to read row by row. */
export const wholeLibrary = { queryKey: getListLibraryQueryKey(), exact: true } as const;
