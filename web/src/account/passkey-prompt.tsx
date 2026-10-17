import { useEffect } from "react";

import { usePasskeySupportSoon } from "../session/passkey";
import { createFlag } from "../shell/flag";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { passkeyPitch, useAddPasskey } from "./add-passkey";
import { DialogActions } from "./dialog-actions";
import { useNeverAsk } from "./key-prompt";

const asked = createFlag();

/** Offers a passkey once after a sign-in whose answer asked for the prompt. */
export function askPasskeyPrompt(): void {
  asked.set(true);
}

function PromptBody({ close }: { close: () => void }) {
  const adding = useAddPasskey(close);
  const never = useNeverAsk(close);
  const failure = adding.failure ?? never.failure;
  const busy = adding.busy || never.busy;
  return (
    <div className="flex flex-col gap-4 text-ink-2">
      <p>{passkeyPitch(location.hostname)}</p>
      {failure !== undefined && <AlertLine text={failure} />}
      <DialogActions>
        <Button
          look="ghost"
          disabled={busy}
          busy={never.busy}
          onClick={() => void never.neverAsk()}
        >
          Don&apos;t ask again
        </Button>
        <Button disabled={busy} onClick={close}>
          Not now
        </Button>
        <Button look="primary" disabled={busy} busy={adding.busy} onClick={() => void adding.add()}>
          Add a passkey
        </Button>
      </DialogActions>
    </div>
  );
}

// Each signed-in frame mounts the prompt, so one asked for at sign-in shows
// in whichever shell the person lands in.
export function PasskeyPrompt() {
  const support = usePasskeySupportSoon();
  const open = asked.useValue() && support?.available === true;
  useEffect(
    () => () => {
      asked.set(false);
    },
    [],
  );
  const close = () => {
    asked.set(false);
  };
  return (
    <Dialog
      title="Sign in faster here with a passkey"
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          close();
        }
      }}
    >
      <PromptBody close={close} />
    </Dialog>
  );
}
