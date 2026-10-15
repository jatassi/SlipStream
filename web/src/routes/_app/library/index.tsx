import { createFileRoute } from "@tanstack/react-router";

import {
  getListContentTypesSuspenseQueryOptions,
  getListWorksSuspenseQueryOptions,
  useListContentTypesSuspense,
  useListWorksSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { Empty } from "../../../shell/empty";
import { Page } from "../../../shell/page";
import { Skeleton } from "../../../ui/skeleton";
import { enabledTypes } from "../../../works/fields";
import { WorkList } from "../../../works/work-list";
import { AddButton, AddMenu } from "./-add-menu";

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
  const { contentTypes } = useListContentTypesSuspense().data;
  const { works } = useListWorksSuspense().data;
  const enabled = enabledTypes(contentTypes);
  return (
    <Page action={enabled.length === 0 ? undefined : <AddMenu types={enabled} />}>
      {works.length > 0 ? (
        <WorkList works={works} />
      ) : enabled.length === 0 ? (
        <Empty>Nothing in the library yet. Turn on a Content Type to add Works.</Empty>
      ) : (
        <Empty
          action={
            <div className="flex flex-wrap gap-2">
              {enabled.map((type) => (
                <AddButton key={type.id} type={type} look="default" />
              ))}
            </div>
          }
        >
          Nothing in the library yet. Add your first Work to start.
        </Empty>
      )}
    </Page>
  );
}
