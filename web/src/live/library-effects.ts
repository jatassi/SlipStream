import { getGetWorkCellsQueryKey, getListWorkFilesQueryKey } from "../api/gen/query/files/files";
import {
  getGetGrabQueryKey,
  getGetManualSearchQueryKey,
  getListBlocklistQueryKey,
  getListGrabsQueryKey,
  getListManualSearchReleasesQueryKey,
  getListSearchesQueryKey,
} from "../api/gen/query/grabs/grabs";
import { getListRootsQueryKey } from "../api/gen/query/roots/roots";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { getGetWorkQueryKey, getListContentTypesQueryKey } from "../api/gen/query/works/works";
import { noteWork } from "./activity";
import { everyPath } from "./refresh";
import { noteRemoved } from "./removed";
import { dropRows, queueRows, wholeLibrary } from "./rows";
import type { LiveEffects } from "./types";

const everyWork = everyPath(/^\/api\/v1\/works\/[^/]+(\/cells)?$/u);
const everyIndexersGrabs = everyPath(/^\/api\/v1\/indexers\/[^/]+\/grabs$/u);
const everyPendingImport = everyPath(/^\/api\/v1\/imports\/pending/u);
const everyManualSearch = everyPath(/^\/api\/v1\/works\/[^/]+\/manual-search(\/releases)?$/u);

/** A Work's Manual search with every pins variant, and its All Releases. */
const manualSearchOf = (workId: string) => {
  const paths = new Set<unknown>([
    getGetManualSearchQueryKey(workId)[0],
    getListManualSearchReleasesQueryKey(workId)[0],
  ]);
  return { predicate: ({ queryKey: [path] }: { queryKey: readonly unknown[] }) => paths.has(path) };
};

/** The counts of files that don't match the naming templates and of files still moving, which renames, imports and naming saves change. */
export const renameCounts = everyPath(/^\/api\/v1\/renames\/(unmatched|moving)$/u);

/**
 * The library's events: Works, their files and the disk queue's moves of
 * them, pending imports, Grabs, searches and the Blocklist. A row of the Library is read again only for the Works an event
 * names. A fetch or a search also moves the Diagram's machines. A Grab's
 * progress tick leaves the cells alone, since a downloading cell reads its
 * progress from the Grab. A Manual search's plan reads the cells, the
 * Grabs holding them and the Blocklist, so each of those refreshes it.
 */
export const libraryEffects = {
  "works.changed": ({ workIds }, { queryClient, refresh }) => {
    noteWork("fetch");
    queueRows(queryClient, workIds);
    for (const workId of workIds) {
      refresh({ queryKey: getGetWorkQueryKey(workId) });
      refresh({ queryKey: getGetWorkCellsQueryKey(workId) });
    }
    refresh({ queryKey: getListRootsQueryKey() });
    refresh(renameCounts);
  },
  "works.removed": ({ workIds }, { queryClient, refresh }) => {
    noteRemoved(queryClient, workIds);
    dropRows(queryClient, workIds);
    refresh({ queryKey: getListRootsQueryKey() });
    refresh({ queryKey: getGetCountsQueryKey() });
  },
  "works.content-types-changed": (_data, { refresh }) => {
    refresh({ queryKey: getListContentTypesQueryKey() });
  },
  "works.type-changed": (_data, { refresh }) => {
    refresh(wholeLibrary);
    refresh(everyWork);
  },
  "files.changed": ({ workId }, { queryClient, refresh }) => {
    queueRows(queryClient, [workId]);
    refresh({ queryKey: getGetWorkCellsQueryKey(workId) });
    refresh({ queryKey: getListWorkFilesQueryKey(workId) });
    refresh({ queryKey: getListRootsQueryKey() });
    refresh(renameCounts);
    refresh(manualSearchOf(workId));
  },
  "files.moved": ({ workIds }, { refresh }) => {
    for (const workId of workIds) {
      refresh({ queryKey: getGetWorkCellsQueryKey(workId) });
      refresh({ queryKey: getListWorkFilesQueryKey(workId) });
      refresh(manualSearchOf(workId));
    }
    refresh(renameCounts);
  },
  "imports.changed": (_data, { refresh }) => {
    refresh(everyPendingImport);
    refresh({ queryKey: getGetCountsQueryKey() });
  },
  "grabs.changed": ({ grabId, workId, progress }, { queryClient, refresh }) => {
    if (!progress) {
      queueRows(queryClient, [workId]);
    }
    refresh({ queryKey: getListGrabsQueryKey() });
    refresh({ queryKey: getGetGrabQueryKey(grabId) });
    refresh(everyIndexersGrabs);
    if (!progress) {
      refresh({ queryKey: getGetWorkCellsQueryKey(workId) });
      refresh(manualSearchOf(workId));
    }
    refresh({ queryKey: getGetCountsQueryKey() });
  },
  "grabs.search-changed": ({ workId, manualSearchId }, { refresh }) => {
    noteWork("search");
    if (manualSearchId === null) {
      refresh({ queryKey: getListSearchesQueryKey(workId) });
    } else {
      refresh(manualSearchOf(workId));
    }
  },
  "grabs.blocklist-changed": (_data, { refresh }) => {
    refresh({ queryKey: getListBlocklistQueryKey() });
    refresh(everyManualSearch);
  },
} satisfies Pick<
  LiveEffects,
  | "works.changed"
  | "works.removed"
  | "works.content-types-changed"
  | "works.type-changed"
  | "files.changed"
  | "files.moved"
  | "imports.changed"
  | "grabs.changed"
  | "grabs.search-changed"
  | "grabs.blocklist-changed"
>;
