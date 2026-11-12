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
import { DownloadEntry, waitingFiles } from "../../../imports/download-entry";
import { Empty } from "../../../shell/empty";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
// oxlint-disable-next-line import/max-dependencies -- the pushed page loads every read a download's Pending import draws from
import { Skeleton } from "../../../ui/skeleton";

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
      title: entry.grab?.release ?? entry.folder,
      trail: [{ title: "Pending imports", pathname: "/library/pending" }],
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Pending import" />,
  component: PendingImport,
});

/** One download's Pending import, the phone's pushed page for it. */
function PendingImport() {
  const entry = useGetImportSuspense(Route.useParams().importId).data;
  const type = useListContentTypesSuspense().data.contentTypes.find(
    ({ id }) => id === entry.contentType,
  );
  return (
    <Page>
      {type === undefined || waitingFiles(entry).length === 0 ? (
        <Empty>Nothing in this download waits for you now.</Empty>
      ) : (
        <ul>
          <DownloadEntry entry={entry} type={type} />
        </ul>
      )}
    </Page>
  );
}
