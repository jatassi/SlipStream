import { createFileRoute } from "@tanstack/react-router";

import {
  getGetImportSuspenseQueryOptions,
  useGetImportSuspense,
} from "../../../api/gen/query/imports/imports";
import { getListNodesSuspenseQueryOptions } from "../../../api/gen/query/nodes/nodes";
import {
  getListQualitiesSuspenseQueryOptions,
  getListTargetsSuspenseQueryOptions,
} from "../../../api/gen/query/targets/targets";
import {
  getGetWorkSuspenseQueryOptions,
  getListContentTypesSuspenseQueryOptions,
  useListContentTypesSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { entryTitle, PendingEntry } from "../../../imports/pending-entry";
import { Page, PagePending } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";

export const Route = createFileRoute("/_app/library/pending_/$importId")({
  staticData: { title: loaderTitle },
  loader: async ({ context: { queryClient }, params: { importId } }) => {
    const [entry] = await Promise.all([
      ensureQuery(queryClient, getGetImportSuspenseQueryOptions(importId)),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListQualitiesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListTargetsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ]);
    if (entry.workId !== null) {
      await ensureQuery(queryClient, getGetWorkSuspenseQueryOptions(entry.workId));
    }
    return {
      title: entryTitle(entry),
      trail: [{ title: "Pending imports", pathname: "/library/pending" }],
    };
  },
  pendingComponent: () => <PagePending label="Loading the Pending import" shape="rows" lines={2} />,
  component: PendingImport,
});

/** One download's or Conversion's Pending import, the phone's pushed page for it. */
function PendingImport() {
  const entry = useGetImportSuspense(Route.useParams().importId).data;
  const type = useListContentTypesSuspense().data.contentTypes.find(
    ({ id }) => id === entry.contentType,
  );
  return (
    <Page>
      <PendingEntry entry={entry} type={type} />
    </Page>
  );
}
