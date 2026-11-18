import { useEffect } from "react";

import { sentenceOf } from "../api/errors";
import type { SubtitlesProvider } from "../api/gen/model";
import { useNow } from "../tasks/use-now";
import { AlertLine } from "../ui/alert-line";
import { Spinner } from "../ui/spinner";
import { toast } from "../ui/toast";
import type { SubtitleSubject } from "./search-host";
import { addedWords, endedWords, stepWords } from "./sheet-words";
import { useAttempt } from "./work-api";
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
  const clock = { now: useNow(60_000), zone: undefined };
  if (error !== null) {
    return <AlertLine text={sentenceOf(error)} />;
  }
  if (attempt === undefined) {
    return null;
  }
  if (attempt.outcome !== null) {
    return attempt.outcome.kind === "added" ? null : (
      <AlertLine text={endedWords(attempt.outcome, { providers, clock })} />
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
