import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { getStatus, getSystem } from "../../api/gen/query/system/system";
import { failAfter, pollInterval, restartStore } from "./restart-state";

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

export type RestoreWait = {
  readonly phase: "waiting" | "back" | "failed";
  readonly retry: () => void;
};

/**
 * A restore's restart, read from the public status, since the restore ends
 * every session and a first-run page has none. `before` is the `startedAt`
 * the Node reported before the restore was confirmed.
 */
export function useRestoreRestart(before: string): RestoreWait {
  const [since, setSince] = useState(() => Date.now());
  const [now, setNow] = useState(since);
  const answer = useQuery({
    queryKey: ["restore-restart", before],
    queryFn: ({ signal }) => getStatus({ signal }),
    refetchInterval: pollInterval,
    retry: false,
    gcTime: 0,
  }).data;
  useEffect(() => {
    const clock = setInterval(() => {
      setNow(Date.now());
    }, pollInterval);
    return () => {
      clearInterval(clock);
    };
  }, []);
  const retry = () => {
    const at = Date.now();
    setSince(at);
    setNow(at);
  };
  if (answer !== undefined && Date.parse(answer.startedAt) > Date.parse(before)) {
    return { phase: "back", retry };
  }
  return { phase: now - since >= failAfter ? "failed" : "waiting", retry };
}
