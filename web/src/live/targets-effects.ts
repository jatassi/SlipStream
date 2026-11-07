import { getListWorkFilesQueryKey } from "../api/gen/query/files/files";
import {
  getListCutsQueryKey,
  getListProfilesQueryKey,
  getListTargetsQueryKey,
} from "../api/gen/query/targets/targets";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import { everyPath } from "./refresh";
import { queueRows, wholeLibrary } from "./rows";
import type { LiveEffects } from "./types";

const everyWorksCells = everyPath(/^\/api\/v1\/works\/[^/]+\/cells$/u);
const everyTarget = everyPath(/^\/api\/v1\/targets\/[^/]+$/u);
const everyProfile = everyPath(/^\/api\/v1\/profiles\/[^/]+$/u);
const everyWorksWants = everyPath(/^\/api\/v1\/works\/[^/]+\/wants$/u);

// Past this many Works a change reads the whole Library once instead of row by row.
const widestSplice = 1000;

/**
 * A change to Targets, Profiles or cuts. The Works it names may have a new
 * Profile and files that fill other cells, or none.
 */
export const targetsEffects = {
  "targets.changed": ({ workIds }, { queryClient, refresh }) => {
    if (workIds.length > widestSplice) {
      refresh(wholeLibrary);
    } else {
      queueRows(queryClient, workIds);
    }
    for (const workId of workIds) {
      refresh({ queryKey: getGetWorkQueryKey(workId) });
      refresh({ queryKey: getListWorkFilesQueryKey(workId) });
    }
    refresh({ queryKey: getListTargetsQueryKey() });
    refresh(everyTarget);
    refresh({ queryKey: getListProfilesQueryKey() });
    refresh(everyProfile);
    refresh({ queryKey: getListCutsQueryKey() });
    refresh(everyWorksWants);
    refresh(everyWorksCells);
  },
} satisfies Pick<LiveEffects, "targets.changed">;
