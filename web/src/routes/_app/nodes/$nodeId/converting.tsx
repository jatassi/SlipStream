import { createFileRoute, notFound } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/nodes/$nodeId/converting")({
  staticData: { title: "Converting" },
  beforeLoad: () => {
    // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
    throw notFound();
  },
});
