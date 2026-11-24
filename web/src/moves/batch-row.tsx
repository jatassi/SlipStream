import { infiniteQueryOptions, useSuspenseInfiniteQuery } from "@tanstack/react-query";
import { Suspense, useState } from "react";

import type { MovesBatchRow } from "../api/gen/model";
import { cancelMoves, getGetMoveQueryKey, getMove } from "../api/gen/query/moves/moves";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import { Spinner } from "../ui/spinner";
import { MoveBackDialog } from "./move-dialog";
import { useRun } from "./run";
import { FailedUnit, UnitLine, beforeSwitch } from "./unit-line";

const pageSize = 100;

const unitPages = (batchId: string) =>
  infiniteQueryOptions({
    queryKey: [...getGetMoveQueryKey(batchId), "pages"],
    queryFn: ({ pageParam, signal }) =>
      getMove(batchId, { after: pageParam, limit: pageSize }, { signal }),
    initialPageParam: -1,
    getNextPageParam: (page) => page.next,
  });

/** The units that haven't switched yet, which a cancel of the batch stops. */
export const remainingOf = ({ counts }: MovesBatchRow): number =>
  beforeSwitch.reduce((sum, phase) => sum + (counts[phase] ?? 0), 0);

/** A batch's units, paged, its failures first in a Needs you band. */
function Units({ batchId }: { batchId: string }) {
  const { data, hasNextPage, isFetchingNextPage, fetchNextPage } = useSuspenseInfiniteQuery(
    unitPages(batchId),
  );
  const units = data.pages.flatMap(({ units: page }) => page);
  const failed = units.filter(({ phase }) => phase === "failed");
  return (
    <div className="mt-2 flex w-full flex-col gap-3">
      {failed.length > 0 && (
        <section aria-label="Needs you" className="flex flex-col gap-2">
          {failed.map((unit) => (
            <FailedUnit key={unit.id} unit={unit} />
          ))}
        </section>
      )}
      <ul className="flex flex-col gap-2">
        {units
          .filter(({ phase }) => phase !== "failed")
          .map((unit) => (
            <UnitLine key={unit.id} unit={unit} />
          ))}
      </ul>
      {hasNextPage && (
        <div>
          <Button
            size="small"
            busy={isFetchingNextPage}
            onClick={() => {
              void fetchNextPage();
            }}
          >
            Show more
          </Button>
        </div>
      )}
    </div>
  );
}

/**
 * One batch in a sentence, such as "12 of 43 movies switched", with its
 * cancel while units haven't switched and Move back once it finished. It
 * expands to each unit.
 */
export function BatchRow({ batch }: { batch: MovesBatchRow }) {
  const { busy, run } = useRun();
  const [back, setBack] = useState(false);
  const remaining = remainingOf(batch);
  return (
    <div className="flex flex-col gap-1 rounded-card bg-glass px-4 py-3 ring-1 ring-hairline ring-inset">
      <Details
        summary={<span className="font-medium">{batch.words}</span>}
        action={
          <span className="flex flex-wrap gap-2">
            {batch.canCancel && remaining > 0 && (
              <Button
                size="small"
                busy={busy === "cancel"}
                onClick={run("cancel", async () => (await cancelMoves(batch.id)).words)}
              >
                {`Cancel the remaining ${remaining}`}
              </Button>
            )}
            {batch.canMoveBack && batch.finishedAt !== undefined && (
              <Button
                size="small"
                onClick={() => {
                  setBack(true);
                }}
              >
                Move back…
              </Button>
            )}
          </span>
        }
      >
        <Suspense fallback={<Spinner label="Loading its Moves" />}>
          <Units batchId={batch.id} />
        </Suspense>
      </Details>
      {back && <MoveBackDialog batchId={batch.id} open onOpenChange={setBack} />}
    </div>
  );
}
