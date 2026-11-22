import { Suspense, useRef } from "react";

import type { NodesOutcome } from "../../../../api/gen/model";
import { drainingOf } from "../../../../moves/drain-plan";
import { type DrainMove, DrainSection } from "../../../../moves/drain-section";
import { Dialog } from "../../../../ui/dialog";
import { SignalLine } from "../../../../ui/signal-line";
import { Checks, Planned } from "./outcome";
import { type Change, type Review, useReview } from "./plan";
import { ReviewFooter } from "./review-footer";

function Body({ review }: { review: Review }) {
  if (review.step === "planning") {
    return <p className="text-ink-2">Checking the change…</p>;
  }
  if (review.step === "unplanned") {
    return (
      <SignalLine tone="fail" announce>
        {review.failure}
      </SignalLine>
    );
  }
  return (
    <div className="flex flex-col gap-4">
      <Planned outcome={review.outcome} stale={review.stale} />
      <Checks checks={review.outcome.checks} />
      {review.failure !== undefined && (
        <SignalLine tone="fail" announce>
          {review.failure}
        </SignalLine>
      )}
    </div>
  );
}

/**
 * The Review: what a change of Roles does, in words, with each guardrail's
 * check, before it is applied. A check that blocks keeps Apply off. A
 * change that drains Storage or Downloading offers to move what the Node
 * keeps, started once the change applies.
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
  const drain = useRef<DrainMove>(null);
  const { review, apply } = useReview(change, async (outcome) => {
    await drain.current?.start();
    await onApplied(outcome);
  });
  const draining = "outcome" in review ? drainingOf(review.outcome) : undefined;
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
      {draining !== undefined && (
        <Suspense fallback={null}>
          <div className="mt-4">
            <DrainSection draining={draining} ref={drain} />
          </div>
        </Suspense>
      )}
      <ReviewFooter review={review} onClose={onClose} onApply={apply} />
    </Dialog>
  );
}
