import { createFileRoute } from "@tanstack/react-router";

import {
  getListContentTypesSuspenseQueryOptions,
  getListWorksSuspenseQueryOptions,
  useListWorksSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { Empty } from "../../../shell/empty";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";
import { enabledTypeOf } from "../../../works/fields";
import { WorkList } from "../../../works/work-list";
import { AddButton } from "./-add-menu";

export const Route = createFileRoute("/_app/library/$contentType")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params: { contentType } }) => {
    const { contentTypes } = await ensureQuery(
      context.queryClient,
      getListContentTypesSuspenseQueryOptions(),
    );
    const type = enabledTypeOf(contentTypes, contentType);
    await ensureQuery(context.queryClient, getListWorksSuspenseQueryOptions({ contentType }));
    return { title: type.work.many, type };
  },
  pendingComponent: () => <Skeleton label="Loading the Library" />,
  component: ContentTypeLibrary,
});

function ContentTypeLibrary() {
  const { type } = Route.useLoaderData();
  const { works } = useListWorksSuspense({ contentType: type.id }).data;
  return (
    <Page action={<AddButton type={type} look="primary" />}>
      {works.length > 0 ? (
        <WorkList works={works} />
      ) : (
        <Empty action={<AddButton type={type} look="default" />}>
          No {type.work.many} in the library yet.
        </Empty>
      )}
    </Page>
  );
}
