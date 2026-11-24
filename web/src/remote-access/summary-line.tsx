import { Link } from "@tanstack/react-router";

import { useGetRemoteAccessSuspense } from "../api/gen/query/reachability/reachability";

/** Nodes' line for Remote access: how many routes work, opening the page. */
export function RemoteAccessLine() {
  const { nothingWanted, routes, status } = useGetRemoteAccessSuspense().data;
  const routeWord = routes.length === 1 ? "route works" : "routes work";
  return (
    <Link
      to="/nodes/remote-access"
      className="rounded-item text-ink-2 underline-offset-4 hover:text-ink-1 hover:underline"
    >
      Remote access ·{" "}
      {nothingWanted ? "Nothing set up" : `${status.counts.work} of ${routes.length} ${routeWord}`}
    </Link>
  );
}
