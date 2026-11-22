import { useState } from "react";

import type { MovesMoveUnit } from "../api/gen/model";
import { cancelMoveUnit, retryMoveUnit } from "../api/gen/query/moves/moves";
import { Button } from "../ui/button";
import { Callout } from "../ui/callout";
import { useRun } from "./run";
import { SwitchNowDialog } from "./switch-now";

export const beforeSwitch = ["queued", "transfer", "place", "switch"] as const;

const canCancel = ({ phase }: MovesMoveUnit) => (beforeSwitch as readonly string[]).includes(phase);

function Actions({ unit, onSwitchNow }: { unit: MovesMoveUnit; onSwitchNow: () => void }) {
  const { busy, run } = useRun();
  return (
    <span className="flex gap-2">
      {unit.switchNow && (
        <Button size="small" onClick={onSwitchNow}>
          Switch now
        </Button>
      )}
      {canCancel(unit) && (
        <Button
          size="small"
          look="ghost"
          busy={busy === "cancel"}
          onClick={run("cancel", async () => {
            await cancelMoveUnit(unit.id);
            return `Canceled the Move of ${unit.name}.`;
          })}
        >
          Cancel
        </Button>
      )}
    </span>
  );
}

/** One unit of a batch: its name, what it does now, its notes, and Switch now or Cancel while they apply. */
export function UnitLine({ unit }: { unit: MovesMoveUnit }) {
  const [switching, setSwitching] = useState(false);
  return (
    <li className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
      <span className="min-w-0">
        <span className="font-medium text-ink-1">{unit.name}</span>{" "}
        <span className="text-ink-2">{unit.words}</span>
        {unit.notes.map((note) => (
          <span key={note} className="block text-meta text-ink-3">
            {note}
          </span>
        ))}
      </span>
      <Actions
        unit={unit}
        onSwitchNow={() => {
          setSwitching(true);
        }}
      />
      {unit.switchNow && (
        <SwitchNowDialog
          unitId={unit.id}
          name={unit.name}
          waiting={unit.waiting}
          open={switching}
          onOpenChange={setSwitching}
        />
      )}
    </li>
  );
}

/** A failed unit in its batch's Needs you band, with Retry. */
export function FailedUnit({ unit }: { unit: MovesMoveUnit }) {
  const { busy, run } = useRun();
  return (
    <Callout
      tone="fail"
      chip="Failed"
      subject={unit.name}
      action={{
        label: "Retry",
        busy: busy === "retry",
        onClick: run("retry", async () => {
          await retryMoveUnit(unit.id);
          return null;
        }),
      }}
    >
      {unit.failure?.words ?? unit.words}
    </Callout>
  );
}
