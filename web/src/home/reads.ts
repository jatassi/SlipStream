import type { QueryClient } from "@tanstack/react-query";

import {
  getGetTorrentCountsSuspenseQueryOptions,
  useGetTorrentCountsSuspense,
} from "../api/gen/query/downloadclients/downloadclients";
import {
  getGetSystemHealthSuspenseQueryOptions,
  getListHealthVolumesSuspenseQueryOptions,
  useGetSystemHealthSuspense,
  useListHealthVolumesSuspense,
} from "../api/gen/query/health/health";
import { getGetHomeSuspenseQueryOptions, useGetHomeSuspense } from "../api/gen/query/home/home";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../api/gen/query/nodes/nodes";
import { getListRootsSuspenseQueryOptions } from "../api/gen/query/roots/roots";
import {
  getGetCountsSuspenseQueryOptions,
  useGetCountsSuspense,
} from "../api/gen/query/shell/shell";
import {
  getListContentTypesSuspenseQueryOptions,
  useListContentTypesSuspense,
} from "../api/gen/query/works/works";
import { ensureQuery } from "../api/query-client";
import { sinceOf } from "../history/day-heading";
// oxlint-disable-next-line import/max-dependencies -- Home's reads are each bounded, and they meet here
import { useClockValue } from "../ui/clock";

/**
 * Everything Home reads before it draws. Each grows with Nodes, roots,
 * Content Types or Health items, never with Grabs, torrents or Works. Roots
 * word a waiting Grab's root.
 */
export const loadHome = (queryClient: QueryClient) =>
  Promise.all([
    ensureQuery(
      queryClient,
      getGetHomeSuspenseQueryOptions({ since: sinceOf("today", new Date()) }),
    ),
    ensureQuery(queryClient, getGetTorrentCountsSuspenseQueryOptions()),
    ensureQuery(queryClient, getGetCountsSuspenseQueryOptions()),
    ensureQuery(queryClient, getGetSystemHealthSuspenseQueryOptions()),
    ensureQuery(queryClient, getListHealthVolumesSuspenseQueryOptions()),
    ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
    ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
  ]);

/** Home's reads. The summary is keyed on the browser's midnight, so a page left open reads today's totals again at midnight. */
export function useHomeReads() {
  const since = useClockValue("minute", (now) => sinceOf("today", now));
  return {
    summary: useGetHomeSuspense({ since }).data,
    counts: useGetCountsSuspense().data,
    report: useGetSystemHealthSuspense().data,
    volumes: useListHealthVolumesSuspense().data.volumes,
    nodes: useListNodesSuspense().data.nodes,
    contentTypes: useListContentTypesSuspense().data.contentTypes,
    counted: useGetTorrentCountsSuspense().data.nodes,
  };
}
