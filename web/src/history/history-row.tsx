import { useState } from "react";

import type { HistoryListEntry } from "../api/gen/model";
import { timeOfDay } from "../format";
import { Button } from "../ui/button";
import { Icon } from "../ui/icon";
import { ListRow } from "../ui/list";
import { Poster } from "../ui/poster";
import { Sheet } from "../ui/sheet";
import { useShell } from "../ui/shell-context";
import { signals } from "../ui/signal-chip";
// oxlint-disable-next-line import/max-dependencies -- a row draws its poster or tone, its words and its long form
import { HistoryEntry } from "./history-entry";

function ToneMark({ tone }: { tone: HistoryListEntry["tone"] }) {
  if (tone !== "fail" && tone !== "warn") {
    return null;
  }
  const { icon } = signals[tone];
  return (
    <Icon
      icon={icon}
      className={`size-4 shrink-0 ${tone === "fail" ? "text-fail-ink" : "text-warn"}`}
    />
  );
}

/** One entry: its time, words and Work's poster. Details opens it in full, in place on desktop and in a sheet on a phone. */
export function HistoryRow({ entry }: { entry: HistoryListEntry }) {
  const phone = useShell() === "phone";
  const [open, setOpen] = useState(false);
  const details = (
    <Button
      look="ghost"
      size="small"
      aria-expanded={phone ? undefined : open}
      onClick={() => {
        setOpen(!open);
      }}
    >
      Details
    </Button>
  );
  return (
    <>
      <ListRow
        leading={
          entry.work === null ? (
            <ToneMark tone={entry.tone} />
          ) : (
            <Poster url={entry.work.posterUrl ?? undefined} />
          )
        }
        title={entry.words}
        meta={timeOfDay(entry.at)}
        trailing={details}
        detail={!phone && open ? <HistoryEntry id={entry.id} /> : undefined}
      />
      {phone && (
        <Sheet title={entry.words} open={open} onOpenChange={setOpen}>
          <HistoryEntry id={entry.id} />
        </Sheet>
      )}
    </>
  );
}
