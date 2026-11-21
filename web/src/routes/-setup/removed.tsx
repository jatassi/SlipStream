import { useSuspenseQuery } from "@tanstack/react-query";

import type { SystemRemoved } from "../../api/gen/model";
import { statusQuery } from "../../session/session";
import { type Clock, useClock } from "../../ui/clock";
import { whenWords } from "../../ui/time-words";

/** S15's sentence for a Node its Controller removed. */
export const removedSentence = ({ controller, at }: SystemRemoved, clock: Clock): string =>
  `This Node was removed from ${controller} ${whenWords(at, "on", clock)}. Join with a new Join Code to use it again.`;

/** The lead of every first-run screen on a Node its Controller removed. */
export function Removed() {
  const { removed } = useSuspenseQuery(statusQuery).data;
  const clock = useClock();
  return removed === undefined ? null : (
    <p className="text-ink-1">{removedSentence(removed, clock)}</p>
  );
}
