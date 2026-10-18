import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import {
  getGetGrabQueryKey,
  getListGrabsQueryKey,
  getListSearchesQueryKey,
} from "../api/gen/query/grabs/grabs";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import {
  getGetWorkQueryKey,
  getListContentTypesQueryKey,
  getListWorksQueryKey,
} from "../api/gen/query/works/works";
import { noteWork } from "./activity";
import { everyPath } from "./refresh";
import type { LiveEffects } from "./types";

const everyCatalogSearch = everyPath(/^\/api\/v1\/content-types\/[^/]+\/catalog$/u);

/** The library's events: Works, their files, Grabs and searches. A fetch or a search also moves the Diagram's machines. */
export const libraryEffects = {
  "works.changed": (data, { refresh }) => {
    noteWork("fetch");
    refresh({ queryKey: getListWorksQueryKey() });
    refresh({ queryKey: getGetWorkQueryKey(data.workId) });
    refresh({ queryKey: getGetWorkCellsQueryKey(data.workId) });
    refresh(everyCatalogSearch);
  },
  "works.content-types-changed": (_data, { refresh }) => {
    refresh({ queryKey: getListContentTypesQueryKey() });
  },
  "files.changed": (data, { refresh }) => {
    refresh({ queryKey: getGetWorkCellsQueryKey(data.workId) });
  },
  "grabs.changed": (data, { refresh }) => {
    refresh({ queryKey: getListGrabsQueryKey() });
    refresh({ queryKey: getGetGrabQueryKey(data.grabId) });
    refresh({ queryKey: getGetWorkCellsQueryKey(data.workId) });
    refresh({ queryKey: getGetCountsQueryKey() });
  },
  "grabs.search-changed": (data, { refresh }) => {
    noteWork("search");
    refresh({ queryKey: getListSearchesQueryKey(data.workId) });
  },
} satisfies Pick<
  LiveEffects,
  | "works.changed"
  | "works.content-types-changed"
  | "files.changed"
  | "grabs.changed"
  | "grabs.search-changed"
>;
