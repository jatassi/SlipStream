import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/activity/torrents")({
  staticData: { title: "Torrents", view: true },
  component: () => null,
});
