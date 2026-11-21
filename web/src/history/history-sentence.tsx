import { useSuspenseQuery } from "@tanstack/react-query";

import { useNow } from "../ui/clock";
import { StatusSentence } from "../ui/status-sentence";
import { summaryOptions } from "./history-query";

/** History's opening sentence: the failed Grabs that need attention, linking to Jobs, else the week's counts. */
export function HistorySentence() {
  const sentence = useSuspenseQuery(summaryOptions(useNow())).data;
  if (sentence === null) {
    return null;
  }
  return (
    <div className="mb-4">
      <StatusSentence
        parts={[
          sentence.needsYou > 0
            ? { kind: "failure", text: sentence.words, href: "/activity" }
            : { kind: "plain", text: sentence.words },
        ]}
      />
    </div>
  );
}
