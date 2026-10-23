import type { HealthCategory } from "../api/gen/model";
import { Details } from "../ui/details";

export function Unread({ category }: { category: HealthCategory }) {
  if (category.unread === undefined) {
    return null;
  }
  return (
    <div className="mb-3">
      <Details
        summary={`SlipStream couldn't read ${category.phrase} just now, so this reads as it last did.`}
      >
        {category.unread}
      </Details>
    </div>
  );
}
