import type { GrabsStopped } from "../api/gen/model";
import { clock } from "../format";
import { useNow } from "../tasks/use-now";
import { Data } from "../ui/data";
import type { FactRow } from "../ui/facts";
import { stoppedWords } from "./search-words";
import type { FailedGrab } from "./stages";

/** A cell failed Grabs stopped SlipStream searching again. */
export type Stop = GrabsStopped;

/** Each failed Grab that stopped the cell, when, and its cause where it isn't the card's, oldest first. */
function Failures({ stop, code }: { stop: Stop; code: FailedGrab["failure"]["code"] }) {
  return (
    <ul className="flex flex-col gap-0.5">
      {stop.failures.map((failure) => (
        <li key={`${failure.at} ${failure.grabId ?? ""}`}>
          {failure.code !== code && `${failure.cause} `}
          <Data>{clock(failure.at)}</Data>
        </li>
      ))}
    </ul>
  );
}

/**
 * A failure card's Recovery: the code's own, or for a stopped cell when
 * Backlog search tries next and each failure that stopped it.
 */
export function useRecoveryFacts(grab: FailedGrab, stop: Stop | undefined): FactRow[] {
  const now = useNow(30_000);
  const { recovery } = grab.failure;
  if (stop !== undefined) {
    return [
      { label: "Recovery", value: stoppedWords(stop, { now, zone: undefined }) },
      { label: "Failures", value: <Failures stop={stop} code={grab.failure.code} /> },
    ];
  }
  return recovery === null ? [] : [{ label: "Recovery", value: recovery }];
}
