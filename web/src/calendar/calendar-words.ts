import type { CalendarSentence } from "../api/gen/model";
import { addDays, betweenWords, monthWords } from "../format/days";
import type { StatusPart } from "../ui/status-sentence";
import type { CalendarPlace } from "./routed";

/** "Nothing in your library comes out in November 2026." */
export const monthEmptyWords = ({ month }: CalendarPlace): string =>
  `Nothing in your library comes out in ${monthWords(month)}.`;

/** "Nothing in your library comes out between 14 November and 26 December." */
export const listEmptyWords = ({ from, to }: CalendarPlace): string =>
  `Nothing in your library comes out between ${betweenWords(from, addDays(to, -1))}.`;

/**
 * The page's opening sentence: what is out and still missing, linked to
 * Wanted, else what comes out this week, then a month's empty words when it
 * has no rows.
 */
export function sentenceParts(
  sentence: CalendarSentence | null,
  empty: string | undefined,
  wanted: string,
): StatusPart[] {
  const lead: StatusPart[] =
    sentence === null
      ? []
      : [
          sentence.needsYou
            ? { kind: "failure", text: sentence.words, href: wanted }
            : { kind: "plain", text: sentence.words },
        ];
  return empty === undefined ? lead : [...lead, { kind: "plain", text: empty }];
}
