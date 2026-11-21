import { useState } from "react";

import { Button } from "../ui/button";
import { type Clock, useClock } from "../ui/clock";
import { Term } from "../ui/term";
import type { RestoreFlow, RestoreStep } from "./flow";
import { NewMachine } from "./new-machine";
import { inPlaceLost, pinSentence, summarySentence } from "./words";
import { SignalLine } from "../ui/signal-line";

function useRemoved() {
  const [removed, setRemoved] = useState<ReadonlySet<string>>(new Set());
  const toggle = (id: string, kept: boolean) => {
    setRemoved((before) => {
      const next = new Set(before);
      if (kept) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };
  return { removed, toggle };
}

function useConfirm(flow: RestoreFlow, step: Extract<RestoreStep, { at: "summary" }>) {
  const [busy, setBusy] = useState(false);
  const restore = async (choice: Parameters<RestoreFlow["confirm"]>[1]) => {
    setBusy(true);
    await flow.confirm(step, choice);
    setBusy(false);
  };
  return { busy, restore };
}

export function InPlaceWords({ takenAt, clock }: { takenAt: string; clock: Clock }) {
  return (
    <p>
      {inPlaceLost(takenAt, clock)} Torrents and files still on your Nodes come back while
      SlipStream is <Term id="reconciling" />. Accounts, PINs and passkeys stay as they are now.
    </p>
  );
}

export function SummaryStep({
  step,
  flow,
  onCancel,
}: {
  step: Extract<RestoreStep, { at: "summary" }>;
  flow: RestoreFlow;
  onCancel: () => void;
}) {
  const clock = useClock();
  const { summary } = step;
  const [off, setOff] = useState(false);
  const ticks = useRemoved();
  const { busy, restore } = useConfirm(flow, step);
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-1">{summarySentence(summary, clock)}</p>
      {summary.newMachine ? (
        <>
          <NewMachine summary={summary} off={off} setOff={setOff} {...ticks} />
          <p>{pinSentence(summary, clock)}</p>
        </>
      ) : (
        <InPlaceWords takenAt={summary.takenAt} clock={clock} />
      )}
      {step.refusal !== undefined && (
        <SignalLine tone="fail" announce>
          {step.refusal}
        </SignalLine>
      )}
      <div className="flex flex-wrap items-center gap-3">
        <Button
          look="primary"
          busy={busy}
          disabled={summary.newMachine && !off}
          onClick={() => void restore({ oldControllerOff: off, remove: [...ticks.removed] })}
        >
          Restore
        </Button>
        <Button look="ghost" disabled={busy} onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </div>
  );
}
