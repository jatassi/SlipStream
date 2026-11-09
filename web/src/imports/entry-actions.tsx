import { useState } from "react";

import type { ImportsEntry, WorksContentType } from "../api/gen/model";
import { Button } from "../ui/button";
import { ignoring, restoring, useAdding } from "./adding";
import { ImportIntoSheet } from "./import-into-sheet";
import { MatchSheet } from "./match-sheet";
import { isAddable, isElsewhere } from "./words";

type Open = "match" | "into";

type ButtonsProps = { entry: ImportsEntry; onOpen: (open: Open) => void };

function OpenButton({
  open,
  onOpen,
  children,
}: {
  open: Open;
  onOpen: (open: Open) => void;
  children: string;
}) {
  return (
    <Button
      size="small"
      onClick={() => {
        onOpen(open);
      }}
    >
      {children}
    </Button>
  );
}

function WaitingButtons({ entry, onOpen }: ButtonsProps) {
  const { add, busy } = useAdding();
  return (
    <>
      {isAddable(entry) && (
        <Button
          size="small"
          look="primary"
          busy={busy}
          onClick={() => {
            add([entry]);
          }}
        >
          Add
        </Button>
      )}
      {!isElsewhere(entry) && (
        <>
          <OpenButton open="match" onOpen={onOpen}>
            Change
          </OpenButton>
          <OpenButton open="into" onOpen={onOpen}>
            Import into
          </OpenButton>
        </>
      )}
      <Button size="small" look="ghost" onClick={() => void ignoring(entry)}>
        Ignore
      </Button>
    </>
  );
}

function IgnoredButtons({ entry, onOpen }: ButtonsProps) {
  return (
    <>
      <Button size="small" onClick={() => void restoring(entry)}>
        Restore
      </Button>
      <OpenButton open="into" onOpen={onOpen}>
        Import into
      </OpenButton>
    </>
  );
}

/**
 * Add, Change, Import into and Ignore; an ignored entry offers Restore and
 * Import into, and one whose Work lives in another root only Ignore.
 */
export function EntryActions({ entry, type }: { entry: ImportsEntry; type: WorksContentType }) {
  const [open, setOpen] = useState<Open>();
  const close = () => {
    setOpen(undefined);
  };
  return (
    <div className="flex flex-wrap gap-2">
      {entry.ignoredAt === null ? (
        <WaitingButtons entry={entry} onOpen={setOpen} />
      ) : (
        <IgnoredButtons entry={entry} onOpen={setOpen} />
      )}
      {open === "match" && <MatchSheet entry={entry} type={type} onClose={close} />}
      {open === "into" && <ImportIntoSheet entry={entry} type={type} onClose={close} />}
    </div>
  );
}
