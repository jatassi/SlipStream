import { Link } from "@tanstack/react-router";
import { Suspense } from "react";

import { useGetRemoteAccessSuspense } from "../api/gen/query/reachability/reachability";

function Summary() {
  const { nothingWanted, routes, status } = useGetRemoteAccessSuspense().data;
  const routeWord = routes.length === 1 ? "route works" : "routes work";
  return (
    <>
      {" · "}
      {nothingWanted ? "Nothing set up" : `${status.counts.work} of ${routes.length} ${routeWord}`}
    </>
  );
}

/** Nodes' line for Remote access: how many routes work, opening the page. It shows before they load. */
export function RemoteAccessLine() {
  return (
    <Link
      to="/nodes/remote-access"
      className="rounded-item text-ink-2 underline-offset-4 hover:text-ink-1 hover:underline"
    >
      Remote access
      <Suspense fallback={null}>
        <Summary />
      </Suspense>
    </Link>
  );
}
