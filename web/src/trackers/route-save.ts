import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { TrackersChoice, TrackersRouteList } from "../api/gen/model";
import { clearTrackerRoute, setTrackerRoute } from "../api/gen/query/trackers/trackers";
import { adoptRoutes } from "../live/adopt-trackers";
import { toast } from "../ui/toast";

/**
 * Saves or drops the client one home Node's Grabs go to. A refusal stays
 * beside the row until the next save.
 */
export function useRouteSave(trackerId: string, homeNodeId: string) {
  const queryClient = useQueryClient();
  const [refusal, setRefusal] = useState<string>();
  const [busy, setBusy] = useState(false);
  const run = async (write: () => Promise<TrackersRouteList>, done: string) => {
    setRefusal(undefined);
    setBusy(true);
    try {
      adoptRoutes(queryClient, trackerId, await write());
      toast(done);
    } catch (error) {
      setRefusal(sentenceOf(error));
    } finally {
      setBusy(false);
    }
  };
  return {
    refusal,
    busy,
    pick: (choice: TrackersChoice, done: string) =>
      run(() => setTrackerRoute(trackerId, homeNodeId, choice), done),
    toDefault: (done: string) => run(() => clearTrackerRoute(trackerId, homeNodeId), done),
  };
}
