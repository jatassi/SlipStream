import { useState } from "react";

import { isProblem } from "../api/errors";
import type { IdentityPasskey } from "../api/gen/model";
import { addPasskey, usePasskeySupport } from "../session/passkey";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { toast } from "../ui/toast";
import { DialogActions } from "./dialog-actions";

// Backing out of the re-confirm dialog is the person's choice, not a failure.
const failureOf = (error: unknown): string | undefined =>
  isProblem(error, "needs_reconfirm")
    ? undefined
    : error instanceof Error
      ? error.message
      : String(error);

/** Runs registration from a click, so every browser lets the ceremony start. */
export function useAddPasskey(onAdded: (passkey: IdentityPasskey) => void) {
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const add = async () => {
    setBusy(true);
    setFailure(undefined);
    try {
      const added = await addPasskey();
      if (added !== null) {
        toast(`Added ${added.name}.`);
        onAdded(added);
      }
    } catch (error) {
      setFailure(failureOf(error));
    } finally {
      setBusy(false);
    }
  };
  return { busy, failure, add };
}

export const passkeyPitch = (host: string): string =>
  `Sign in at ${host} with Face ID, Touch ID or your device's screen lock instead of your PIN.`;

function AddPasskeyBody({ onDone }: { onDone: () => void }) {
  const support = usePasskeySupport();
  const { busy, failure, add } = useAddPasskey(onDone);
  return (
    <div className="flex flex-col gap-4 text-ink-2">
      <p>{support.available ? passkeyPitch(location.hostname) : support.why}</p>
      {failure !== undefined && <AlertLine text={failure} />}
      <DialogActions>
        <Button disabled={busy} onClick={onDone}>
          {support.available ? "Cancel" : "Close"}
        </Button>
        {support.available && (
          <Button look="primary" busy={busy} onClick={() => void add()}>
            Continue
          </Button>
        )}
      </DialogActions>
    </div>
  );
}

export function AddPasskeyDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title="Add a passkey" open={open} onOpenChange={onOpenChange}>
      <AddPasskeyBody
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}
