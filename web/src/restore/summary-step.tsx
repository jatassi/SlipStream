import { useState } from "react";

import type { Clock } from "../format";
import { useNow } from "../tasks/use-now";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Term } from "../ui/term";
import type { RestoreFlow, RestoreStep } from "./flow";
import { NewMachine } from "./new-machine";
import { inPlaceLost, pinSentence, summarySentence } from "./words";

const minute = 60_000;

export const useRestoreClock = (): Clock => ({ now: useNow(minute), zone: undefined });

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
  const clock = useRestoreClock();
  const { summary } = step;
  const [off, setOff] = useState(false);
  const ticks = useRemoved();
  const [busy, setBusy] = useState(false);
  const restore = async () => {
    setBusy(true);
    await flow.confirm(step, { oldControllerOff: off, remove: [...ticks.removed] });
    setBusy(false);
  };
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
      {step.refusal !== undefined && <AlertLine text={step.refusal} />}
      <div className="flex flex-wrap items-center gap-3">
        <Button
          look="primary"
          busy={busy}
          disabled={summary.newMachine && !off}
          onClick={() => void restore()}
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
