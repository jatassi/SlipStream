import { useState } from "react";

import type { ConversionsConversionRow } from "../api/gen/model";
import { openManualSearch } from "../manual/sheet-state";
import { Button } from "../ui/button";
import { SignalRow } from "../ui/signal-row";
import { retry } from "./actions";
import { Attempts } from "./attempts";
import { CancelDialog } from "./cancel-dialogs";
import { listed, subjectOf } from "./row-parts";

function failureVerdict(row: ConversionsConversionRow): string {
  const of = `The ${row.target.name} Conversion of ${row.work.title}`;
  if (row.failure?.class === "source") {
    return `${of} can't run with the ${row.source.target} file.`;
  }
  const nodes = row.failure?.nodes ?? [];
  return nodes.length === 0 ? `${of} failed.` : `${of} failed on ${listed.format(nodes)}.`;
}

export function AttentionRow({ row }: { row: ConversionsConversionRow }) {
  const [canceling, setCanceling] = useState(false);
  const subject = subjectOf(row);
  const ended = row.attempts.at(-1)?.endedAt ?? undefined;
  return (
    <SignalRow
      signal="fail"
      chip="Failed"
      subject={subject}
      verdict={failureVerdict(row)}
      {...(ended !== undefined && { at: ended })}
    >
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap gap-2">
          <Button
            look="on-fail"
            onClick={() => {
              void retry(row, subject);
            }}
          >
            Retry
          </Button>
          <Button
            look="on-fail"
            onClick={() => {
              openManualSearch({
                workId: row.work.id,
                scope: { itemIds: row.items.map((item) => item.id) },
              });
            }}
          >
            Search for a new {row.source.target} file
          </Button>
          <Button
            look="on-fail"
            onClick={() => {
              setCanceling(true);
            }}
          >
            Cancel…
          </Button>
        </div>
        <Attempts row={row} />
      </div>
      <CancelDialog row={row} open={canceling} onOpenChange={setCanceling} />
    </SignalRow>
  );
}
