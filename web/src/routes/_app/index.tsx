import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/")({
  beforeLoad: () => {
    redirect({ to: "/library", replace: true, throw: true });
  },
});
