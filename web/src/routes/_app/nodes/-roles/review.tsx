import type { NodesOutcome } from "../../../../api/gen/model";
import { blocks } from "../../../../nodes/roles";
import { AlertLine } from "../../../../ui/alert-line";
import { Button } from "../../../../ui/button";
import { Dialog } from "../../../../ui/dialog";
import { useShell } from "../../../../ui/shell-context";
import { Checks, Planned } from "./outcome";
import { type Change, type Review, useReview } from "./plan";

function Body({ review }: { review: Review }) {
  if (review.step === "planning") {
    return <p className="text-ink-2">Checking the change…</p>;
  }
  if (review.step === "unplanned") {
    return <AlertLine text={review.failure} />;
  }
  return (
    <div className="flex flex-col gap-4">
      <Planned outcome={review.outcome} stale={review.stale} />
      <Checks checks={review.outcome.checks} />
      {review.failure !== undefined && <AlertLine text={review.failure} />}
    </div>
  );
}

/**
 * The Review: what a change of Roles does, in words, with each guardrail's
 * check, before it is applied. A check that blocks keeps Apply off.
 */
export function ReviewDialog({
  change,
  onClose,
  onApplied,
}: {
  change: Change;
  onClose: () => void;
  onApplied: (outcome: NodesOutcome) => Promise<void>;
}) {
  const { review, apply } = useReview(change, onApplied);
  const stacked = useShell() === "phone";
  return (
    <Dialog
      title="Review the change"
      size={680}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <div aria-live="polite">
        <Body review={review} />
      </div>
      <div className={`mt-6 flex gap-2 ${stacked ? "flex-col-reverse" : "justify-end"}`}>
        <Button disabled={review.step === "applying"} onClick={onClose}>
          Cancel
        </Button>
        {"outcome" in review && (
          <Button
            look="primary"
            busy={review.step === "applying"}
            disabled={blocks(review.outcome).length > 0}
            onClick={() => void apply(review)}
          >
            Apply
          </Button>
        )}
      </div>
    </Dialog>
  );
}
