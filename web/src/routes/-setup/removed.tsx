import { useSuspenseQuery } from "@tanstack/react-query";

import type { SystemRemoved } from "../../api/gen/model";
import { statusQuery } from "../../session/session";

const day = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short" });

/** S15's sentence for a Node its Controller removed, as the server logs it. */
export const removedSentence = ({ controller, at }: SystemRemoved): string =>
  `This Node was removed from ${controller} on ${day.format(new Date(at))}. Join with a new Join Code to use it again.`;

/** The lead of every first-run screen on a Node its Controller removed. */
export function Removed() {
  const { removed } = useSuspenseQuery(statusQuery).data;
  return removed === undefined ? null : <p className="text-ink-1">{removedSentence(removed)}</p>;
}
