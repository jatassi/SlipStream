import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { z } from "zod";

import { BlocklistBrowse } from "../../../blocklist/browse";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/activity/blocklist")({
  staticData: { title: "Blocklist", view: true },
  validateSearch: z.object({ offset: z.number().int().min(1).optional().catch(undefined) }),
  pendingComponent: () => <Skeleton label="Loading the Blocklist" />,
  component: ActivityBlocklist,
});

function ActivityBlocklist() {
  const navigate = useNavigate();
  return (
    <BlocklistBrowse
      search={Route.useSearch()}
      onSearchChange={(change) => {
        void navigate({
          to: "/activity/blocklist",
          search: (prev) => ({ ...prev, ...change }),
          replace: true,
        });
      }}
    />
  );
}
