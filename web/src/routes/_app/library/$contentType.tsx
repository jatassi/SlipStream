import { Add01Icon } from "@hugeicons/core-free-icons";
import { Link, createFileRoute } from "@tanstack/react-router";

import {
  getListContentTypesSuspenseQueryOptions,
  getListWorksSuspenseQueryOptions,
  useListWorksSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { loaderTitle } from "../../../shell/titles";
import { Page } from "../../../shell/page";
import { Button } from "../../../ui/button";
import { Skeleton } from "../../../ui/skeleton";
import { addLabel, enabledTypeOf } from "../../../works/fields";
import { WorkList } from "../../../works/work-list";

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
    <Page
      action={
        <Button
          look="primary"
          icon={Add01Icon}
          nativeButton={false}
          render={<Link to="/add" search={{ contentType: type.id }} />}
        >
          {addLabel(type)}
        </Button>
      }
    >
      <WorkList works={works} />
    </Page>
  );
}
