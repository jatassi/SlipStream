import { createFileRoute, notFound } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/activity/conversions")({
  staticData: { title: "Conversions", view: true },
  beforeLoad: () => {
    // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
    throw notFound();
  },
});
