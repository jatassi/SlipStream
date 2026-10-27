import type { QueryClient } from "@tanstack/react-query";

import {
  getGetWorkCellsSuspenseQueryOptions,
  getListWorkFilesSuspenseQueryOptions,
} from "../../api/gen/query/files/files";
import {
  getListRootsSuspenseQueryOptions,
  getListVolumesSuspenseQueryOptions,
} from "../../api/gen/query/roots/roots";
import {
  getGetWorkWantsSuspenseQueryOptions,
  getListCutsSuspenseQueryOptions,
  getListProfilesSuspenseQueryOptions,
  getListQualitiesSuspenseQueryOptions,
  getListTargetsSuspenseQueryOptions,
} from "../../api/gen/query/targets/targets";
import {
  getGetWorkSuspenseQueryOptions,
  getListContentTypesSuspenseQueryOptions,
} from "../../api/gen/query/works/works";
import { ensureQuery } from "../../api/query-client";
import { loadWorkGrabs } from "../../grabs/waiting";
import type { Crumb } from "../../shell/titles";
import { libraryTrail } from "../fields";

/** Every read the Work page draws from, so it opens whole; the title and crumbs for the shell. */
export async function loadWork(
  queryClient: QueryClient,
  workId: string,
): Promise<{ title: string; trail: Crumb[] }> {
  const [{ contentTypes }, work] = await Promise.all([
    ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
    ensureQuery(queryClient, getGetWorkSuspenseQueryOptions(workId)),
    ensureQuery(queryClient, getGetWorkCellsSuspenseQueryOptions(workId)),
    ensureQuery(queryClient, getListWorkFilesSuspenseQueryOptions(workId)),
    ensureQuery(queryClient, getGetWorkWantsSuspenseQueryOptions(workId)),
    loadWorkGrabs(queryClient, workId),
    ensureQuery(queryClient, getListProfilesSuspenseQueryOptions()),
    ensureQuery(queryClient, getListCutsSuspenseQueryOptions()),
    ensureQuery(queryClient, getListTargetsSuspenseQueryOptions()),
    ensureQuery(queryClient, getListQualitiesSuspenseQueryOptions()),
    ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
    ensureQuery(queryClient, getListVolumesSuspenseQueryOptions()),
  ]);
  return {
    title: work.label,
    trail: libraryTrail(contentTypes, work.contentType),
  };
}
