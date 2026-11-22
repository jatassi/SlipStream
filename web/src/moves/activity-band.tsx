import type { MovesBatchRow } from "../api/gen/model";
import { Section } from "../ui/section";
import { BatchRow } from "./batch-row";

/** Activity's Moves: one row per batch, the open ones first. */
export function MovesSection({ batches }: { batches: readonly MovesBatchRow[] }) {
  if (batches.length === 0) {
    return null;
  }
  const open = batches.filter(({ finishedAt }) => finishedAt === undefined).length;
  return (
    <Section title="Moves" {...(open > 0 && { count: open })}>
      <div className="flex flex-col gap-2">
        {batches.map((batch) => (
          <BatchRow key={batch.id} batch={batch} />
        ))}
      </div>
    </Section>
  );
}
