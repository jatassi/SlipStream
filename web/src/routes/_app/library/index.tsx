import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";

import { useListContentTypesSuspense } from "../../../api/gen/query/works/works";
import { RoutedLibrary, librarySearch, rememberedSearch } from "../../../library/routed";
import { Empty } from "../../../shell/empty";
import { Page } from "../../../shell/page";
import { Skeleton } from "../../../ui/skeleton";
import { enabledTypes } from "../../../works/fields";
import { AddButton, AddMenu } from "./-add-menu";
import { libraryReads } from "./-view";

const scope = "all";

export const Route = createFileRoute("/_app/library/")({
  staticData: { title: "Library" },
  validateSearch: librarySearch,
  beforeLoad: ({ search }) => {
    const remembered = rememberedSearch(scope, search);
    if (remembered !== undefined) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router follows a thrown redirect()
      throw redirect({ to: "/library", search: remembered, replace: true });
    }
  },
  loader: ({ context }) => libraryReads(context.queryClient),
  pendingComponent: () => <Skeleton label="Loading the Library" />,
  component: Library,
});

function Library() {
  const navigate = useNavigate();
  const { contentTypes } = useListContentTypesSuspense().data;
  const enabled = enabledTypes(contentTypes);
  return (
    <Page action={enabled.length === 0 ? undefined : <AddMenu types={enabled} />}>
      <RoutedLibrary
        scope={scope}
        types={enabled}
        search={Route.useSearch()}
        onSearch={(search) => {
          void navigate({ to: "/library", search, replace: true });
        }}
        empty={
          enabled.length === 0 ? (
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
          )
        }
      />
    </Page>
  );
}
