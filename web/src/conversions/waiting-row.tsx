import { useState } from "react";

import type { ConversionsConversionRow } from "../api/gen/model";
import { useClock } from "../ui/clock";
import { useShell } from "../ui/shell-context";
import { moveUp } from "./actions";
import { CancelDialog } from "./cancel-dialogs";
import { Handle, MoveUpButton, useDrop } from "./moving";
import { Lines, RowMenu, WorkLink, rowLook, subjectOf } from "./row-parts";
import { waitingLine } from "./words";

export function WaitingRow({ row }: { row: ConversionsConversionRow }) {
  const clock = useClock();
  const desk = useShell() !== "phone";
  const [canceling, setCanceling] = useState(false);
  const { over, handlers } = useDrop(row);
  const subject = subjectOf(row);
  const movable = row.stage === "queued";
  const up = () => {
    void moveUp(row, subject);
  };
  return (
    <div className={`${rowLook} ${over ? "ring-solid" : ""}`} {...handlers}>
      {movable && desk && <Handle row={row} />}
      <Lines first={<WorkLink row={row} />} second={waitingLine(row, clock)} />
      {movable && desk && <MoveUpButton subject={subject} onMoveUp={up} />}
      <RowMenu
        label={subject}
        items={[
          ...(movable ? [{ label: "Move up", onSelect: up }] : []),
          {
            label: "Cancel…",
            tone: "danger",
            onSelect: () => {
              setCanceling(true);
            },
          },
        ]}
      />
      <CancelDialog row={row} open={canceling} onOpenChange={setCanceling} />
    </div>
  );
}
