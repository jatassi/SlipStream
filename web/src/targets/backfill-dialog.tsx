import { useState } from "react";

import { DialogActions } from "../account/dialog-actions";
import type { TargetsBackfill } from "../api/gen/model";
import { Button } from "../ui/button";
import { Callout } from "../ui/callout";
import { Dialog } from "../ui/dialog";
import { type Attempt, useAttempt } from "./attempt";
import {
  type Backfilling,
  type Shown,
  backfillOf,
  convertedOf,
  firstPlan,
  restating,
  stepsOf,
  unnamed,
} from "./backfill-plan";
import { CostStep, RestStep, StepList, WorksStep } from "./backfill-steps";

function StepActions({
  at,
  last,
  saving,
  blocked,
  onAt,
  onClose,
}: {
  at: number;
  last: boolean;
  saving: Attempt;
  blocked: boolean;
  onAt: (at: number) => void;
  onClose: () => void;
}) {
  return (
    <DialogActions>
      <Button disabled={saving.busy} onClick={onClose}>
        Cancel
      </Button>
      {at > 0 && (
        <Button
          disabled={saving.busy}
          onClick={() => {
            onAt(at - 1);
          }}
        >
          Back
        </Button>
      )}
      {last ? (
        <Button look="primary" busy={saving.busy} disabled={blocked} onClick={saving.run}>
          Save
        </Button>
      ) : (
        <Button
          look="primary"
          onClick={() => {
            onAt(at + 1);
          }}
        >
          Next
        </Button>
      )}
    </DialogActions>
  );
}

type Props = {
  readonly shown: Shown;
  readonly backfilling: Backfilling;
  readonly onSave: (open: Backfilling, backfill: TargetsBackfill, made: number) => Promise<void>;
  readonly onClose: () => void;
};

function BackfillBody({ shown, backfilling, onSave, onClose }: Props) {
  const { impact, changed } = backfilling;
  const steps = stepsOf(shown.noun);
  const [at, setAt] = useState(0);
  const [plan, setPlan] = useState(() => firstPlan(shown));
  const converted = convertedOf(plan, impact);
  const saving = useAttempt(() => onSave(backfilling, backfillOf(plan, impact), converted.files));
  const step = { plan, impact, shown, onPlan: setPlan };
  const last = at === steps.length - 1;
  return (
    <div className="flex flex-col gap-4">
      <StepList steps={steps} at={at} />
      {changed && (
        <Callout tone="warn" chip="Changed">
          These counts changed since the last preview. Check them before you save.
        </Callout>
      )}
      {at === 0 && <CostStep {...step} />}
      {at === 1 && <WorksStep {...step} />}
      {last && (
        <>
          <RestStep {...step} rest={converted.rest} />
          <output className="text-ink-1">{restating(plan, impact, shown)}</output>
        </>
      )}
      {saving.failure}
      <StepActions
        at={at}
        last={last}
        saving={saving}
        blocked={unnamed(plan, impact)}
        onAt={setAt}
        onClose={onClose}
      />
    </div>
  );
}

/**
 * Asks what a Profile save that makes files for Works it already has should
 * do: what it costs, which Works to convert, and where the rest go.
 */
export function BackfillDialog(
  props: Omit<Props, "backfilling" | "shown"> & {
    readonly shown: Shown | undefined;
    readonly backfilling: Backfilling | undefined;
  },
) {
  const { shown, backfilling } = props;
  return (
    <Dialog
      title={
        shown === undefined
          ? "Convert existing Works"
          : `Convert existing ${shown.noun.many} to ${shown.made}`
      }
      size={940}
      open={backfilling !== undefined}
      onOpenChange={(open) => {
        if (!open) {
          props.onClose();
        }
      }}
    >
      {backfilling !== undefined && shown !== undefined && (
        <BackfillBody
          key={backfilling.changed ? "changed" : "first"}
          {...props}
          shown={shown}
          backfilling={backfilling}
        />
      )}
    </Dialog>
  );
}
