import type { SupersededView } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { whenWords } from "../ui/time-words";

const sentenceStart = (words: string): string => words.charAt(0).toUpperCase() + words.slice(1);

/** The page's lead: when and how this Controller was replaced, and what stopped. */
export function leadOf(view: SupersededView, clock: Clock): string {
  const when = view.at === undefined ? "" : `${sentenceStart(whenWords(view.at, "at", clock))}, `;
  if (view.state === "proven" && view.replacement !== undefined) {
    return `${when}a Backup of this library was restored on the Controller at ${view.replacement.address}. This Controller's torrents are paused and its searches have stopped.`;
  }
  return `${sentenceStart(`${when}an Admin stopped this Controller.`)} Its torrents are paused and its searches have stopped.`;
}
