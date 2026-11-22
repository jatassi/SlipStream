import { createFileRoute } from "@tanstack/react-router";

import { HomePage } from "../../home/home-page";
import { loadHome } from "../../home/reads";
import { PagePending } from "../../shell/page";

export const Route = createFileRoute("/_app/")({
  staticData: { title: "Home" },
  loader: ({ context: { queryClient } }) => loadHome(queryClient),
  pendingComponent: () => <PagePending label="Loading Home" shape="rows" lines={2} />,
  component: HomePage,
});
