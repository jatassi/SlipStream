import { useEffect, useRef } from "react";

import { useRestoreRestart } from "../shell/restart/restart-poll";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Spinner } from "../ui/spinner";
import { restartFailedSentence, restartingSentence } from "./words";

export function RestartingStep({ before, onBack }: { before: string; onBack: () => void }) {
  const wait = useRestoreRestart(before);
  const back = wait.phase === "back";
  const handed = useRef(false);
  useEffect(() => {
    if (back && !handed.current) {
      handed.current = true;
      onBack();
    }
  }, [back, onBack]);
  if (wait.phase === "failed") {
    return (
      <div className="flex flex-col items-start gap-3">
        <AlertLine text={restartFailedSentence} />
        <Button onClick={wait.retry}>Try again</Button>
      </div>
    );
  }
  return (
    <output className="flex items-center gap-2 text-ink-1">
      <Spinner className="size-4 text-ink-2" />
      {restartingSentence}
    </output>
  );
}
