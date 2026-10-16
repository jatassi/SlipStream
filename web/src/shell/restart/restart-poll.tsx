import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import { getSystem } from "../../api/gen/query/system/system";
import { pollInterval, restartStore } from "./restart-state";

// Each answer from the browser's Node goes to the store, which reloads the
// page once a later process answers. A Node that is down answers nothing,
// so errors are expected and only the clock moves the wait on.
export function WaitForSelf() {
  const answer = useQuery({
    queryKey: ["restart-poll"],
    queryFn: ({ signal }) => getSystem({ signal }),
    refetchInterval: pollInterval,
    retry: false,
    gcTime: 0,
  }).data;
  useEffect(() => {
    if (answer !== undefined) {
      restartStore.dispatch({ type: "answered", startedAt: answer.startedAt });
    }
  }, [answer]);
  useEffect(() => {
    const clock = setInterval(() => {
      restartStore.dispatch({ type: "tick", at: Date.now() });
    }, pollInterval);
    return () => {
      clearInterval(clock);
    };
  }, []);
  return null;
}
