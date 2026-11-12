import { Link, createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { getListImportsSuspenseQueryOptions } from "../../../api/gen/query/imports/imports";
import { getListNodesSuspenseQueryOptions } from "../../../api/gen/query/nodes/nodes";
import {
  getListQualitiesSuspenseQueryOptions,
  getListTargetsSuspenseQueryOptions,
} from "../../../api/gen/query/targets/targets";
import {
  getListContentTypesSuspenseQueryOptions,
  getListLibrarySuspenseQueryOptions,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { PendingPage, type PendingView, pendingParams } from "../../../imports/pending-page";
import { SegmentedLinks, segmentedLink } from "../../../ui/segmented-links";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/library/pending")({
  staticData: { title: "Pending imports" },
  validateSearch: z.object({ view: z.literal("ignored").optional().catch(undefined) }),
  loaderDeps: ({ search: { view } }): { view: PendingView } => ({ view: view ?? "needs-you" }),
  loader: async ({ context: { queryClient }, deps: { view } }) => {
    await Promise.all([
      ensureQuery(queryClient, getListImportsSuspenseQueryOptions(pendingParams(view))),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListQualitiesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListTargetsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListLibrarySuspenseQueryOptions()),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading Pending imports" />,
  component: Pending,
});

const exact = { exact: true, includeSearch: true };

function ViewSwitch() {
  return (
    <SegmentedLinks label="Pending imports views">
      <Link to="/library/pending" search={{}} activeOptions={exact} className={segmentedLink}>
        Needs you
      </Link>
      <Link
        to="/library/pending"
        search={{ view: "ignored" }}
        activeOptions={exact}
        className={segmentedLink}
      >
        Ignored
      </Link>
    </SegmentedLinks>
  );
}

function Pending() {
  const view: PendingView = Route.useSearch().view ?? "needs-you";
  return <PendingPage view={view} beside={<ViewSwitch />} />;
}
