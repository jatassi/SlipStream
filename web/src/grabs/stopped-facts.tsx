import type { ReactNode } from "react";

import type { GrabsStopped } from "../api/gen/model";
import { clock } from "../format";
import { useNow } from "../tasks/use-now";
import { Data } from "../ui/data";
import { stoppedWords } from "./search-words";
import { type FailedGrab, failureWords } from "./stages";

/** A cell failed Grabs stopped SlipStream searching again. */
export type Stop = GrabsStopped;

/** Each failed Grab that stopped the cell, its cause and when, oldest first. */
function Failures({ stop }: { stop: Stop }) {
  return (
    <ul className="flex flex-col gap-0.5">
      {stop.failures.map((failure) => (
        <li key={`${failure.at} ${failure.grabId ?? ""}`}>
          {failureWords[failure.code].cause} <Data>{clock(failure.at)}</Data>
        </li>
      ))}
    </ul>
  );
}

/**
 * A failure card's Recovery: the code's own, or for a stopped cell when
 * Backlog search tries next and each failure that stopped it.
 */
export function useRecoveryFacts(
  grab: FailedGrab,
  stop: Stop | undefined,
): { label: string; value: ReactNode }[] {
  const now = useNow(30_000);
  const { recovery } = failureWords[grab.failure.code];
  if (stop !== undefined) {
    return [
      { label: "Recovery", value: stoppedWords(stop, { now, zone: undefined }) },
      { label: "Failures", value: <Failures stop={stop} /> },
    ];
  }
  return recovery === undefined ? [] : [{ label: "Recovery", value: recovery }];
}
