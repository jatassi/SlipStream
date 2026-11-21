import { useEffect } from "react";

import { sentenceOf } from "../api/errors";
import type { SubtitlesProvider } from "../api/gen/model";
import { useClock } from "../ui/clock";
import { SignalLine } from "../ui/signal-line";
import { Spinner } from "../ui/spinner";
import { toast } from "../ui/toast";
import type { SubtitleSubject } from "./search-host";
import { addedWords, endedWords, stepWords } from "./sheet-words";
import { useAttempt } from "./work-api";
// oxlint-disable-next-line import/max-dependencies -- the line follows an attempt's steps and outcome, and says why its read failed
import { providerName } from "./work-words";

/**
 * Follows one pick or upload through its steps. Once its subtitle is
 * added it toasts and closes the sheet.
 */
export function useAttemptLine(
  subject: SubtitleSubject,
  attemptId: string | undefined,
  onAdded: () => void,
) {
  const read = useAttempt(subject.fileId, attemptId);
  const attempt = read.data;
  const added = attempt?.outcome?.kind === "added";
  useEffect(() => {
    if (added) {
      toast(addedWords(subject.wanted));
      onAdded();
    }
  }, [added, subject.wanted, onAdded]);
  return { attempt, error: read.error };
}

/** What an attempt is doing, how it ended without a subtitle, or why it can't be read. */
export function AttemptLine({
  line: { attempt, error },
  providers,
}: {
  line: ReturnType<typeof useAttemptLine>;
  providers: readonly SubtitlesProvider[];
}) {
  const clock = useClock();
  if (error !== null) {
    return (
      <SignalLine tone="fail" announce>
        {sentenceOf(error)}
      </SignalLine>
    );
  }
  if (attempt === undefined) {
    return null;
  }
  if (attempt.outcome !== null) {
    return attempt.outcome.kind === "added" ? null : (
      <SignalLine tone="fail" announce>
        {endedWords(attempt.outcome, { providers, clock })}
      </SignalLine>
    );
  }
  const words =
    attempt.step === "downloading"
      ? stepWords.downloading(providerName(providers, attempt.provider ?? ""))
      : stepWords.checking;
  return (
    <output className="flex items-center gap-2 text-ink-2">
      <Spinner />
      {words}
    </output>
  );
}
