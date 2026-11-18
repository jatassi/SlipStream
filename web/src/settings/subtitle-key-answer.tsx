import { Tick02Icon } from "@hugeicons/core-free-icons";

import { DialogActions } from "../account/dialog-actions";
import type { SubtitlesProvider } from "../api/gen/model";
import type { KeyAnswer } from "../subtitles/settings-api";
import { acceptedWords } from "../subtitles/settings-words";
import { Button } from "../ui/button";
import { DialogClose } from "../ui/dialog";
import { Icon } from "../ui/icon";
import { SignalChip } from "../ui/signal-chip";

function Accepted({ provider }: { provider: SubtitlesProvider }) {
  return (
    <div className="flex flex-col gap-4">
      <output className="flex items-start gap-3 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset">
        <span className="flex size-7.5 shrink-0 items-center justify-center rounded-capsule bg-solid text-on-solid shadow-live-glow">
          <Icon icon={Tick02Icon} className="size-4" />
        </span>
        <span className="pt-1 text-ink-1">{acceptedWords(provider)}</span>
      </output>
      <DialogActions>
        <DialogClose>
          <Button look="primary">Done</Button>
        </DialogClose>
      </DialogActions>
    </div>
  );
}

function Refused({ words, onRetry }: { words: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col gap-4">
      <output className="flex flex-col items-start gap-3 rounded-card bg-fail-card px-5 py-4 text-fail-text shadow-signal-lift">
        <SignalChip kind="fail">Not saved</SignalChip>
        <span>{words}</span>
      </output>
      <DialogActions>
        <DialogClose>
          <Button>Cancel</Button>
        </DialogClose>
        <Button look="primary" onClick={onRetry}>
          Try another key
        </Button>
      </DialogActions>
    </div>
  );
}

/** The provider's answer, which replaces the steps: a result card with a glowing check, or its words in a failure card. */
export function KeyAnswerCard({ answer, onRetry }: { answer: KeyAnswer; onRetry: () => void }) {
  return answer.kind === "accepted" ? (
    <Accepted provider={answer.provider} />
  ) : (
    <Refused words={answer.words} onRetry={onRetry} />
  );
}
