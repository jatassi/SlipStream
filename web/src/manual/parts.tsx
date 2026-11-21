import { CheckmarkCircle02Icon } from "@hugeicons/core-free-icons";

import type { GrabsEditionOverride, GrabsPickResult, GrabsWordedReason } from "../api/gen/model";
import { Button } from "../ui/button";
import { Callout } from "../ui/callout";
import { Icon } from "../ui/icon";
import { SignalLine } from "../ui/signal-line";

function Override({ words }: { words: string }) {
  return (
    <Callout tone="warn" chip="Override">
      {words}
    </Callout>
  );
}

/** What a pick gets past and the Edition it is grabbed as, amber: a soft refusal is a choice to make, not a fault. */
export function Overrides({
  lifts,
  edition,
}: {
  lifts: readonly GrabsWordedReason[];
  edition: GrabsEditionOverride | null;
}) {
  return (
    <>
      {lifts.map((lift) => (
        <Override key={`${lift.code}/${lift.targetId ?? ""}`} words={lift.words} />
      ))}
      {edition !== null && <Override words={edition.words} />}
    </>
  );
}

/** What a grab did for one pick: grabbed, or its problem, with a cancel offer for a held cell. */
export function Result({
  result,
  busy,
  onCancelHeld,
}: {
  result: GrabsPickResult;
  busy: boolean;
  onCancelHeld: (grabId: string) => void;
}) {
  const { problem } = result;
  if (problem === null) {
    return (
      <p className="flex items-center gap-1.5 text-ink-1">
        <Icon icon={CheckmarkCircle02Icon} className="size-4 shrink-0 text-ok" />
        Grabbed.
      </p>
    );
  }
  const held = problem.code === "cell_held" ? problem.grabId : undefined;
  return (
    <div className="flex flex-col items-start gap-2">
      <SignalLine tone="fail" announce>
        {problem.detail}
      </SignalLine>
      {held !== undefined && (
        <Button
          size="small"
          busy={busy}
          onClick={() => {
            onCancelHeld(held);
          }}
        >
          Cancel it and grab this one
        </Button>
      )}
    </div>
  );
}
