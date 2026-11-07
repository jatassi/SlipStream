import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import { runTask } from "../../api/gen/query/tasks/tasks";
import { Button } from "../../ui/button";
import { toast } from "../../ui/toast";

/** Runs the Indexer catalog refresh now. */
export function RefreshNow() {
  const [busy, setBusy] = useState(false);
  return (
    <Button
      busy={busy}
      onClick={() => {
        setBusy(true);
        runTask("indexer-catalog-refresh")
          .catch((error: unknown) => {
            toast(sentenceOf(error), { kind: "fail" });
          })
          .finally(() => {
            setBusy(false);
          });
      }}
    >
      Refresh now
    </Button>
  );
}

/** The catalog before its first download: when SlipStream retries, and Refresh now. */
export function NotYet({ notice }: { notice: string }) {
  return (
    <div className="flex flex-col items-start gap-3 rounded-card bg-glass p-5 ring-1 ring-hairline ring-inset">
      <p className="text-ink-1">{notice}</p>
      <RefreshNow />
    </div>
  );
}
