import { type ReactElement, useState } from "react";

import type { SubtitlesProvider } from "../api/gen/model";
import { Dialog } from "../ui/dialog";
import type { KeyAnswer } from "../subtitles/settings-api";
import { KeyAnswerCard } from "./subtitle-key-answer";
import { KeySteps } from "./subtitle-key-steps";

/**
 * The three guided steps to a provider's free key. A pasted key is checked
 * at once, and the provider's answer replaces the steps.
 */
function SubtitleKeyDialog({
  provider,
  open,
  onOpenChange,
}: {
  provider: SubtitlesProvider;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [answer, setAnswer] = useState<KeyAnswer>();
  return (
    <Dialog
      title={`Add your ${provider.name} key`}
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          setAnswer(undefined);
        }
        onOpenChange(next);
      }}
    >
      {answer === undefined ? (
        <KeySteps provider={provider} onAnswer={setAnswer} />
      ) : (
        <KeyAnswerCard
          answer={answer}
          onRetry={() => {
            setAnswer(undefined);
          }}
        />
      )}
    </Dialog>
  );
}

/**
 * One key Dialog for the section, opened for a provider. It stays mounted
 * while the button that opened it goes, as the lead's does once a key is
 * accepted.
 */
export function useKeyDialog(): {
  readonly openKey: (provider: SubtitlesProvider) => void;
  readonly dialog: ReactElement | null;
} {
  const [provider, setProvider] = useState<SubtitlesProvider>();
  const [open, setOpen] = useState(false);
  return {
    openKey: (chosen) => {
      setProvider(chosen);
      setOpen(true);
    },
    dialog:
      provider === undefined ? null : (
        <SubtitleKeyDialog provider={provider} open={open} onOpenChange={setOpen} />
      ),
  };
}
