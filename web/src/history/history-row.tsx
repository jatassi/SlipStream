import { useState } from "react";

import type { HistoryListEntry } from "../api/gen/model";
import { Chevron } from "../ui/chevron";
import { Icon } from "../ui/icon";
import { ListRow } from "../ui/list";
import { PlainTime } from "../ui/relative-time";
import { Artwork } from "../ui/artwork";
import { Sheet } from "../ui/sheet";
import { useShell } from "../ui/shell-context";
import { signals } from "../ui/signal-chip";
// oxlint-disable-next-line import/max-dependencies -- a row draws its poster, its signal, its words and its long form
import { HistoryEntry } from "./history-entry";

const toneInk = { fail: "text-fail-ink", warn: "text-warn" } as const;

/** The time of day, led by the fail or warn icon when the entry is one. */
function Meta({ entry }: { entry: HistoryListEntry }) {
  const { tone } = entry;
  return (
    <>
      {(tone === "fail" || tone === "warn") && (
        <Icon
          icon={signals[tone].icon}
          className={`mr-1 inline-block size-3.5 align-[-2px] ${toneInk[tone]}`}
        />
      )}
      <PlainTime at={entry.at} form="time" />
    </>
  );
}

/** The poster column every row keeps, so each row's words start at one edge whether or not it has a poster. */
function Leading({ entry }: { entry: HistoryListEntry }) {
  const phone = useShell() === "phone";
  if (entry.work === null) {
    return <span aria-hidden="true" className={`shrink-0 ${phone ? "w-8" : "w-6"}`} />;
  }
  return (
    <Artwork use="poster" size={phone ? "row" : "option"} url={entry.work.posterUrl ?? undefined} />
  );
}

function Disclosure({
  words,
  open,
  onOpenChange,
}: {
  words: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <button
      type="button"
      aria-expanded={open}
      aria-label={`Details, ${words}`}
      {...(open && { "data-panel-open": "" })}
      onClick={() => {
        onOpenChange(!open);
      }}
      className="group -mr-2 inline-flex h-(--control-height) items-center gap-1 rounded-item px-2 text-body font-medium text-ink-2 hover:text-ink-1"
    >
      <Chevron />
      Details
    </button>
  );
}

type RowProps = {
  entry: HistoryListEntry;
  /** Left out on the Work page, where every row is that Work's. */
  poster?: boolean;
};

function PhoneRow({ entry, poster = true }: RowProps) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <ListRow
        leading={poster ? <Leading entry={entry} /> : undefined}
        title={entry.words}
        wrap
        meta={<Meta entry={entry} />}
        trailing={<Chevron />}
        render={
          // oxlint-disable-next-line jsx-a11y/control-has-associated-label -- ListRow fills the button with the entry's words and time
          <button
            type="button"
            onClick={() => {
              setOpen(true);
            }}
          />
        }
      />
      <Sheet title={entry.words} open={open} onOpenChange={setOpen}>
        <HistoryEntry id={entry.id} />
      </Sheet>
    </>
  );
}

function DesktopRow({ entry, poster = true }: RowProps) {
  const [open, setOpen] = useState(false);
  return (
    <ListRow
      leading={poster ? <Leading entry={entry} /> : undefined}
      title={entry.words}
      meta={<Meta entry={entry} />}
      trailing={<Disclosure words={entry.words} open={open} onOpenChange={setOpen} />}
      detail={
        open ? (
          <div className={poster ? "pl-9" : undefined}>
            <HistoryEntry id={entry.id} />
          </div>
        ) : undefined
      }
    />
  );
}

/** One entry: its words, its time and, on the History page, its Work's poster. Desktop opens it in full in place, and a phone's whole row opens it in a sheet. */
export function HistoryRow(props: RowProps) {
  return useShell() === "phone" ? <PhoneRow {...props} /> : <DesktopRow {...props} />;
}
