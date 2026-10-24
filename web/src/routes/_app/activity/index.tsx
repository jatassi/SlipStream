import { createFileRoute, useNavigate, useSearch } from "@tanstack/react-router";

import { getListTorrentsSuspenseQueryOptions } from "../../../api/gen/query/downloadclients/downloadclients";
import { getListGrabsSuspenseQueryOptions } from "../../../api/gen/query/grabs/grabs";
import {
  getListQualitiesSuspenseQueryOptions,
  getListTargetsSuspenseQueryOptions,
} from "../../../api/gen/query/targets/targets";
import {
  getListContentTypesSuspenseQueryOptions,
  getListWorksSuspenseQueryOptions,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { Jobs } from "../../../grabs/jobs";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/activity/")({
  staticData: { title: "Activity" },
  loader: ({ context: { queryClient } }) =>
    Promise.all([
      ensureQuery(queryClient, getListGrabsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListWorksSuspenseQueryOptions()),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListTorrentsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListTargetsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListQualitiesSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading Jobs" />,
  component: JobsPage,
});

function JobsPage() {
  const navigate = useNavigate();
  const { type } = useSearch({ from: "/_app/activity" });
  return (
    <Jobs
      type={type}
      onTypeChange={(next) => {
        void navigate({ to: ".", search: (prev) => ({ ...prev, type: next }), replace: true });
      }}
    />
  );
}
