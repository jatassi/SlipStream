import type { QueryClient } from "@tanstack/react-query";
import { z } from "zod";

import { getListTorrentsSuspenseQueryOptions } from "../api/gen/query/downloadclients/downloadclients";
import { getListNodesSuspenseQueryOptions } from "../api/gen/query/nodes/nodes";
import {
  getListCutsSuspenseQueryOptions,
  getListQualitiesSuspenseQueryOptions,
} from "../api/gen/query/targets/targets";
import { getListContentTypesSuspenseQueryOptions } from "../api/gen/query/works/works";
import { ensureQuery } from "../api/query-client";
import { ensureLibrary } from "../library/projection";

/** The search a Torrents page keeps in its address. */
export const torrentsSearch = z.object({
  tab: z
    .enum(["downloading", "seeding", "problems", "goal-met", "all"])
    .optional()
    .catch(undefined),
});

/** What a Torrents page reads before it draws. */
export const loadTorrents = (queryClient: QueryClient) =>
  Promise.all([
    ensureQuery(queryClient, getListTorrentsSuspenseQueryOptions()),
    ensureLibrary(queryClient),
    ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
    ensureQuery(queryClient, getListQualitiesSuspenseQueryOptions()),
    ensureQuery(queryClient, getListCutsSuspenseQueryOptions()),
  ]);
