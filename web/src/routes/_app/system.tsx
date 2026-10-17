import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/system")({
  beforeLoad: () => {
    redirect({ to: "/nodes", replace: true, throw: true });
  },
});
