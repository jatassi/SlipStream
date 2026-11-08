import { createFileRoute } from "@tanstack/react-router";

import { loaderTitle } from "../../../../../shell/titles";

export const Route = createFileRoute("/_app/settings/trackers/$trackerId/_torrents/")({
  staticData: { title: loaderTitle },
  component: () => null,
});
