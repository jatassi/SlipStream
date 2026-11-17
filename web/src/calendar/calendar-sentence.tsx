import type { CalendarAnswer } from "../api/gen/model";
import { StatusSentence } from "../ui/status-sentence";
import { monthEmptyWords, sentenceParts } from "./calendar-words";
import type { CalendarPlace } from "./routed";

/** What is out and still missing, linked to Wanted, else what comes out this week, then an empty month's words. */
export function CalendarSentence({
  answer,
  place,
  contentType,
}: {
  answer: CalendarAnswer;
  place: CalendarPlace;
  contentType: string | undefined;
}) {
  const parts = sentenceParts(
    answer.sentence,
    place.view === "month" && answer.days.length === 0 ? monthEmptyWords(place) : undefined,
    contentType === undefined ? "/library/wanted" : `/library/wanted/${contentType}`,
  );
  if (parts.length === 0) {
    return null;
  }
  return (
    <div className="mb-4">
      <StatusSentence parts={parts} />
    </div>
  );
}
