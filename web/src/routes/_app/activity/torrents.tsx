import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { getListTorrentsSuspenseQueryOptions } from "../../../api/gen/query/downloadclients/downloadclients";
import { getListNodesSuspenseQueryOptions } from "../../../api/gen/query/nodes/nodes";
import {
  getListCutsSuspenseQueryOptions,
  getListQualitiesSuspenseQueryOptions,
} from "../../../api/gen/query/targets/targets";
import { getListContentTypesSuspenseQueryOptions } from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { ensureLibrary } from "../../../library/projection";
import { TorrentsPage } from "../../../torrents/torrents-page";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/activity/torrents")({
  staticData: { title: "Torrents", view: true },
  validateSearch: z.object({
    tab: z.enum(["downloading", "seeding", "problems", "all"]).optional().catch(undefined),
  }),
  loader: ({ context }) =>
    Promise.all([
      ensureQuery(context.queryClient, getListTorrentsSuspenseQueryOptions()),
      ensureLibrary(context.queryClient),
      ensureQuery(context.queryClient, getListNodesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListQualitiesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListCutsSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading Torrents" />,
  component: TorrentsPage,
});
