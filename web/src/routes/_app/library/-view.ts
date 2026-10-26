import type { QueryClient } from "@tanstack/react-query";

import {
  getListRootsSuspenseQueryOptions,
  getListVolumesSuspenseQueryOptions,
} from "../../../api/gen/query/roots/roots";
import { getListProfilesSuspenseQueryOptions } from "../../../api/gen/query/targets/targets";
import { getListContentTypesSuspenseQueryOptions } from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { ensureLibrary } from "../../../library/projection";

/** Everything a Library reads before it draws, the bulk dialogs' volumes included. */
export const libraryReads = (queryClient: QueryClient) =>
  Promise.all([
    ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
    ensureLibrary(queryClient),
    ensureQuery(queryClient, getListProfilesSuspenseQueryOptions()),
    ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
    ensureQuery(queryClient, getListVolumesSuspenseQueryOptions()),
  ]);
