import { CheckmarkCircle02Icon } from "@hugeicons/core-free-icons";

import type {
  GrabsEditionOverride,
  GrabsPickResult,
  GrabsReleaseDetails,
  GrabsWordedReason,
} from "../api/gen/model";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Chip } from "../ui/chip";
import { Icon } from "../ui/icon";
import { SignalChip } from "../ui/signal-chip";

export function ReleaseName({ release }: { release: GrabsReleaseDetails }) {
  return (
    <span className="flex flex-wrap items-center gap-2 text-ink-1">
      {release.words}
      {release.freeleech && <Chip>Freeleech</Chip>}
    </span>
  );
}

function Override({ words }: { words: string }) {
  return (
    <p className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-card bg-warn-card px-3 py-2 text-warn-text ring-1 ring-warn-line ring-inset">
      <SignalChip kind="warn">Override</SignalChip>
      <span className="min-w-0 flex-[1_1_24ch]">{words}</span>
    </p>
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
      <AlertLine text={problem.detail} />
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
