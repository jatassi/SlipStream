import { createFileRoute } from "@tanstack/react-router";

import {
  getListContentTypesSuspenseQueryOptions,
  getListWorksSuspenseQueryOptions,
  useListContentTypesSuspense,
  useListWorksSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { Page } from "../../../shell/page";
import { Skeleton } from "../../../ui/skeleton";
import { enabledTypes } from "../../../works/fields";
import { WorkList } from "../../../works/work-list";
import { AddMenu } from "./-add-menu";

export const Route = createFileRoute("/_app/library/")({
  staticData: { title: "Library" },
  loader: ({ context }) =>
    Promise.all([
      ensureQuery(context.queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListWorksSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading the Library" />,
  component: Library,
});

function Library() {
  const types = enabledTypes(useListContentTypesSuspense().data.contentTypes);
  const { works } = useListWorksSuspense().data;
  return (
    <Page action={types.length === 0 ? undefined : <AddMenu types={types} />}>
      <WorkList works={works} />
    </Page>
  );
}
