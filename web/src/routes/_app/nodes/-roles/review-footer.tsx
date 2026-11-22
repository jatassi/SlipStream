import { blocks } from "../../../../nodes/roles";
import { Button } from "../../../../ui/button";
import { useShell } from "../../../../ui/shell-context";
import type { Ready, Review } from "./plan";

/** The Review's Cancel and Apply; a check that blocks keeps Apply off. */
export function ReviewFooter({
  review,
  onClose,
  onApply,
}: {
  review: Review;
  onClose: () => void;
  onApply: (review: Ready) => Promise<void>;
}) {
  const stacked = useShell() === "phone";
  return (
    <div className={`mt-6 flex gap-2 ${stacked ? "flex-col-reverse" : "justify-end"}`}>
      <Button disabled={review.step === "applying"} onClick={onClose}>
        Cancel
      </Button>
      {"outcome" in review && (
        <Button
          look="primary"
          busy={review.step === "applying"}
          disabled={blocks(review.outcome).length > 0}
          onClick={() => void onApply(review)}
        >
          Apply
        </Button>
      )}
    </div>
  );
}
