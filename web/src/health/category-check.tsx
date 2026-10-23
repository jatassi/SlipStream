import type { HealthCategory } from "../api/gen/model";
import { useCheckHealthCategory } from "../api/gen/query/health/health";
import { Button } from "../ui/button";
import { Checked, useCheckAnswered } from "./checked";

/** The category's Check now, and its answer once there is one. */
export function useCategoryCheck(category: HealthCategory) {
  const check = useCheckHealthCategory({ mutation: useCheckAnswered() });
  return {
    button: category.canCheck ? (
      <Button
        size="small"
        busy={check.isPending}
        aria-label={`Check now: ${category.label}`}
        onClick={() => {
          check.mutate({ category: category.id });
        }}
      >
        Check now
      </Button>
    ) : undefined,
    answer: check.data !== undefined && (
      <div className="mb-3 text-ink-2">
        <Checked unreached={check.data.unreached} />
      </div>
    ),
  };
}
