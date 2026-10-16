import { type ErrorComponentProps, getRouteApi, useRouter } from "@tanstack/react-router";
import { useEffect, useState } from "react";

import { isProblem } from "../../../api/errors";
import { Empty } from "../../../shell/empty";
import { RouteError } from "../../../shell/errors";
import { stateOfLink, useNode } from "./-node";

const nodeRoute = getRouteApi("/_app/nodes/$nodeId");

// A section that failed because its Node went offline loads again once the
// Node's link has been seen down and then back up.
function useReloadWhenBack(online: boolean, reload: () => void) {
  const [seenDown, setSeenDown] = useState(!online);
  if (!online && !seenDown) {
    setSeenDown(true);
  }
  useEffect(() => {
    if (online && seenDown) {
      reload();
    }
  }, [online, seenDown, reload]);
}

/** A section's error: an offline Node says so in a sentence, anything else is the route's error card. */
export function NodeProblem(props: ErrorComponentProps) {
  const router = useRouter();
  const node = useNode(nodeRoute.useParams().nodeId);
  useReloadWhenBack(stateOfLink[node.link.state] === "online", () => {
    props.reset();
    void router.invalidate();
  });
  if (!isProblem(props.error, "node_offline")) {
    return <RouteError {...props} />;
  }
  return <Empty>{node.name} is offline. Its settings and logs come back when it reconnects.</Empty>;
}
