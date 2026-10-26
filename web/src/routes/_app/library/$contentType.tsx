import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";

import { useListContentTypesSuspense } from "../../../api/gen/query/works/works";
import { RoutedLibrary, librarySearch, rememberedSearch } from "../../../library/routed";
import { Empty } from "../../../shell/empty";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";
import { enabledTypeOf } from "../../../works/fields";
import { AddButton } from "./-add-menu";
import { libraryReads } from "./-view";

export const Route = createFileRoute("/_app/library/$contentType")({
  staticData: { title: loaderTitle },
  validateSearch: librarySearch,
  beforeLoad: ({ search, params: { contentType } }) => {
    const remembered = rememberedSearch(contentType, search);
    if (remembered !== undefined) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router follows a thrown redirect()
      throw redirect({
        to: "/library/$contentType",
        params: { contentType },
        search: remembered,
        replace: true,
      });
    }
  },
  loader: async ({ context, params: { contentType } }) => {
    const [{ contentTypes }] = await libraryReads(context.queryClient);
    return { title: enabledTypeOf(contentTypes, contentType).work.many };
  },
  pendingComponent: () => <Skeleton label="Loading the Library" />,
  component: ContentTypeLibrary,
});

function ContentTypeLibrary() {
  const navigate = useNavigate();
  const { contentType } = Route.useParams();
  const type = enabledTypeOf(useListContentTypesSuspense().data.contentTypes, contentType);
  return (
    <Page action={<AddButton type={type} look="primary" />}>
      <RoutedLibrary
        scope={type.id}
        types={[type]}
        search={Route.useSearch()}
        onSearch={(search) => {
          void navigate({
            to: "/library/$contentType",
            params: { contentType },
            search,
            replace: true,
          });
        }}
        empty={
          <Empty action={<AddButton type={type} look="default" />}>
            No {type.work.many} in the library yet.
          </Empty>
        }
      />
    </Page>
  );
}
