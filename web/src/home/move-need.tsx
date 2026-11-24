import { useState } from "react";

import type { HomeMoveNeed } from "../api/gen/model";
import { retryMoveUnit } from "../api/gen/query/moves/moves";
import { useRun } from "../moves/run";
import { SwitchNowDialog } from "../moves/switch-now";
import { Callout } from "../ui/callout";

function FailedMove({ need }: { need: HomeMoveNeed }) {
  const { busy, run } = useRun();
  return (
    <Callout
      tone="fail"
      chip="Move failed"
      subject={need.work.label}
      action={{
        label: "Retry",
        busy: busy === "retry",
        onClick: run("retry", async () => {
          await retryMoveUnit(need.unitId);
          return null;
        }),
      }}
    >
      {need.words}
    </Callout>
  );
}

function WaitingMove({ need }: { need: HomeMoveNeed }) {
  const [switching, setSwitching] = useState(false);
  return (
    <>
      <Callout
        tone="warn"
        chip="Waiting to switch"
        subject={need.work.label}
        {...(need.since !== undefined && { since: need.since })}
        {...(need.switchNow && {
          action: {
            label: "Switch now",
            onClick: () => {
              setSwitching(true);
            },
          },
        })}
      >
        {need.words}
      </Callout>
      {need.switchNow && (
        <SwitchNowDialog
          unitId={need.unitId}
          name={need.work.label}
          waiting={undefined}
          open={switching}
          onOpenChange={setSwitching}
        />
      )}
    </>
  );
}

/** Moves that need the Admin: each failure with Retry, each long switch wait with Switch now. */
export function MoveNeeds({ needs }: { needs: readonly HomeMoveNeed[] }) {
  return needs.map((need) =>
    need.kind === "failed" ? (
      <FailedMove key={need.unitId} need={need} />
    ) : (
      <WaitingMove key={need.unitId} need={need} />
    ),
  );
}
