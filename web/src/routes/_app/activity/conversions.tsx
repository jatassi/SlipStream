import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { ConversionsPage } from "../../../conversions/conversions-page";
import { conversionsSearch, loadConversions, paramsOf } from "../../../conversions/load";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/activity/conversions")({
  staticData: { title: "Conversions", view: true },
  validateSearch: conversionsSearch,
  loaderDeps: ({ search: { band, node, type, q } }) => ({ band, node, type, q }),
  loader: ({ context: { queryClient }, deps }) => loadConversions(queryClient, paramsOf(deps)),
  pendingComponent: () => <Skeleton label="Loading Conversions" shape="rows" lines={2} />,
  component: ActivityConversions,
});

function ActivityConversions() {
  const navigate = useNavigate();
  return (
    <ConversionsPage
      search={Route.useSearch()}
      onSearchChange={(change) => {
        void navigate({ to: ".", search: (prev) => ({ ...prev, ...change }), replace: true });
      }}
    />
  );
}
