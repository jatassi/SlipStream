import { Segmented } from "../ui/segmented";
import type { HoldDraft, HoldUnit, TargetDraft } from "./draft";
import { holdProblem } from "./draft";
import type { SectionProps } from "./editor";
import { DraftField } from "./field";

const units: readonly { value: HoldUnit; label: string }[] = [
  { value: "minutes", label: "minutes" },
  { value: "hours", label: "hours" },
  { value: "days", label: "days" },
];

/** A Target's Hold: how long an automatic grab waits for a better Release, 0 to 7 days. */
export function HoldField({ draft, onChange }: SectionProps<TargetDraft>) {
  const set = (hold: HoldDraft) => {
    onChange({ ...draft, hold });
  };
  return (
    <div className="flex flex-wrap items-start gap-3">
      <div className="w-60">
        <DraftField
          label="Wait for a better release for"
          value={draft.hold.amount}
          inputMode="decimal"
          error={holdProblem(draft.hold)}
          description="A Release that meets the cutoff and the upgrade-until score is grabbed at once."
          onChange={(amount) => {
            set({ ...draft.hold, amount });
          }}
        />
      </div>
      <div className="pt-7">
        <Segmented
          label="Hold unit"
          options={units}
          value={draft.hold.unit}
          onValueChange={(unit) => {
            set({ ...draft.hold, unit });
          }}
        />
      </div>
    </div>
  );
}
