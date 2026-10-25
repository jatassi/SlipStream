import type { HealthCategory } from "../api/gen/model";
import { Details } from "../ui/details";

export function Unread({ category }: { category: HealthCategory }) {
  if (category.unread === undefined) {
    return null;
  }
  return (
    <div className="mb-3">
      <Details
        summary={`SlipStream couldn't read ${category.phrase} just now. This shows the last known state.`}
      >
        {category.unread}
      </Details>
    </div>
  );
}
