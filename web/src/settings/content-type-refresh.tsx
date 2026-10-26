import { useRouter } from "@tanstack/react-router";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { MetadataRefreshAllRun, WorksContentType } from "../api/gen/model";
import {
  refreshContentType,
  useGetContentTypeRefreshSuspense,
} from "../api/gen/query/metadata/metadata";
import { clock } from "../format";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";

function lastRunWords({ lastRunAt, running }: MetadataRefreshAllRun) {
  if (running) {
    return "It's running now.";
  }
  return lastRunAt === null ? "It hasn't run yet." : `It last ran ${clock(lastRunAt)}.`;
}

export function RefreshAll({ type }: { type: WorksContentType }) {
  const router = useRouter();
  const run = useGetContentTypeRefreshSuspense(type.id).data;
  const [starting, setStarting] = useState(false);
  const start = async () => {
    setStarting(true);
    try {
      await refreshContentType(type.id);
      toast(`Refreshing every ${type.work.one} from ${type.source}.`);
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
        Refresh all reads every {type.work.one} from {type.source} again. Use it after changing the
        metadata language or region. {lastRunWords(run)}
      </p>
      <Button busy={starting} disabled={run.running} onClick={() => void start()}>
        Refresh all
      </Button>
    </div>
  );
}
