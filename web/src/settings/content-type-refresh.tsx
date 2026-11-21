import { useRouter } from "@tanstack/react-router";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { MetadataRefreshAllRun, WorksContentType } from "../api/gen/model";
import {
  refreshContentType,
  useGetContentTypeRefreshSuspense,
} from "../api/gen/query/metadata/metadata";
import { Button } from "../ui/button";
import { type Clock, useClock } from "../ui/clock";
import { whenWords } from "../ui/time-words";
import { toast } from "../ui/toast";

function lastRunWords({ lastRunAt, running }: MetadataRefreshAllRun, clock: Clock) {
  if (running) {
    return "It's running now.";
  }
  return lastRunAt === null
    ? "It hasn't run yet."
    : `It last ran ${whenWords(lastRunAt, "at", clock)}.`;
}

export function RefreshAll({ type }: { type: WorksContentType }) {
  const router = useRouter();
  const run = useGetContentTypeRefreshSuspense(type.id).data;
  const clock = useClock();
  const [starting, setStarting] = useState(false);
  const start = async () => {
    setStarting(true);
    try {
      const { runId } = await refreshContentType(type.id);
      toast(
        runId === null
          ? `Metadata refresh is already running. The next run refreshes every ${type.work.one.toLowerCase()} from ${type.source}.`
          : `Refreshing every ${type.work.one.toLowerCase()} from ${type.source}.`,
      );
      await router.invalidate();
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setStarting(false);
    }
  };
  return (
    <div className="flex flex-col items-start gap-2">
      <p className="text-ink-2">
        Refresh all fetches the details of every {type.work.one.toLowerCase()} from {type.source}{" "}
        again. Use it after changing the metadata language or region. {lastRunWords(run, clock)}
      </p>
      <Button busy={starting} disabled={run.running} onClick={() => void start()}>
        Refresh all
      </Button>
    </div>
  );
}
