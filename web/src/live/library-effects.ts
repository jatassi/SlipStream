import { getGetWorkCellsQueryKey, getListWorkFilesQueryKey } from "../api/gen/query/files/files";
import {
  getGetGrabQueryKey,
  getListGrabsQueryKey,
  getListSearchesQueryKey,
} from "../api/gen/query/grabs/grabs";
import { getListRootsQueryKey } from "../api/gen/query/roots/roots";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import {
  getGetWorkQueryKey,
  getListContentTypesQueryKey,
  getListWorksQueryKey,
} from "../api/gen/query/works/works";
import { noteWork } from "./activity";
import { everyPath } from "./refresh";
import { noteRemoved } from "./removed";
import { dropRows, queueRows, wholeLibrary } from "./rows";
import type { LiveEffects } from "./types";

const everyWork = everyPath(/^\/api\/v1\/works(\/[^/]+(\/cells)?)?$/u);

/**
 * The library's events: Works, their files, Grabs and searches. A row of
 * the Library is read again only for the Works an event names. A fetch or
 * a search also moves the Diagram's machines.
 */
export const libraryEffects = {
  "works.changed": ({ workIds }, { queryClient, refresh }) => {
    noteWork("fetch");
    queueRows(queryClient, workIds);
    refresh({ queryKey: getListWorksQueryKey() });
    for (const workId of workIds) {
      refresh({ queryKey: getGetWorkQueryKey(workId) });
      refresh({ queryKey: getGetWorkCellsQueryKey(workId) });
    }
    refresh({ queryKey: getListRootsQueryKey() });
  },
  "works.removed": ({ workIds }, { queryClient, refresh }) => {
    noteRemoved(queryClient, workIds);
    dropRows(queryClient, workIds);
    refresh({ queryKey: getListWorksQueryKey() });
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
  },
  "grabs.changed": ({ grabId, workId, progress }, { queryClient, refresh }) => {
    if (!progress) {
      queueRows(queryClient, [workId]);
    }
    refresh({ queryKey: getListGrabsQueryKey() });
    refresh({ queryKey: getGetGrabQueryKey(grabId) });
    refresh({ queryKey: getGetWorkCellsQueryKey(workId) });
    refresh({ queryKey: getGetCountsQueryKey() });
  },
  "grabs.search-changed": (data, { refresh }) => {
    noteWork("search");
    refresh({ queryKey: getListSearchesQueryKey(data.workId) });
  },
} satisfies Pick<
  LiveEffects,
  | "works.changed"
  | "works.removed"
  | "works.content-types-changed"
  | "works.type-changed"
  | "files.changed"
  | "grabs.changed"
  | "grabs.search-changed"
>;
