import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import type { NodesNode } from "../../../../api/gen/model";
import {
  getGetSupersededQueryKey,
  getSuperseded,
  stopSupersededController,
} from "../../../../api/gen/query/superseded/superseded";
import { Button } from "../../../../ui/button";
import { ConfirmDialog } from "../../../../ui/confirm-dialog";
import { Section } from "../../../../ui/section";

/**
 * Stop this Controller, on the page of the Worker that said another
 * Controller replaced it and carried no proof, which its Health Error opens.
 */
export function StopRow({ node }: { node: NodesNode }) {
  const [asking, setAsking] = useState(false);
  const fence = useQuery({
    queryKey: getGetSupersededQueryKey(),
    queryFn: ({ signal }) => getSuperseded({ signal }),
  }).data;
  if (fence?.state !== "unproven" || fence.workerId !== node.id) {
    return null;
  }
  return (
    <Section title="Another Controller">
      <div className="flex flex-col items-start gap-3">
        <p className="max-w-[640px] text-ink-2">
          {node.name} now connects to a different Controller and sent no proof of a restore. If you
          restored a Backup on another machine, stop this Controller.
        </p>
        <Button
          look="fail"
          onClick={() => {
            setAsking(true);
          }}
        >
          Stop this Controller…
        </Button>
      </div>
      <ConfirmDialog
        title="Stop this Controller?"
        confirmLabel="Stop"
        look="fail"
        open={asking}
        onOpenChange={setAsking}
        onConfirm={() => stopSupersededController()}
      >
        <p>
          This Controller pauses every torrent, stops searching and opens only the page that wipes
          it and joins the new Controller. Resume this Controller on that page undoes it.
        </p>
      </ConfirmDialog>
    </Section>
  );
}
