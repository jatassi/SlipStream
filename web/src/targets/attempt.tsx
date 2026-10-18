import { type ReactNode, useState } from "react";

import { DialogActions } from "../account/dialog-actions";
import { isApiError, sentenceOf } from "../api/errors";
import { AlertLine } from "../ui/alert-line";
import { Button, type ButtonLook } from "../ui/button";

// A save can refuse with a sentence of its own, as a ConfirmDialog's action can.
export const refusalOf = (error: unknown): string =>
  error instanceof Error && !isApiError(error) ? error.message : sentenceOf(error);

export type Attempt = {
  readonly busy: boolean;
  readonly failure: ReactNode;
  readonly run: () => void;
};

export function useAttempt(action: () => Promise<unknown>): Attempt {
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const attempt = async () => {
    setBusy(true);
    setFailure(undefined);
    try {
      await action();
    } catch (error) {
      setFailure(refusalOf(error));
    } finally {
      setBusy(false);
    }
  };
  return {
    busy,
    failure: failure === undefined ? null : <AlertLine text={failure} />,
    run: () => void attempt(),
  };
}

export function AttemptActions({
  attempt,
  look = "primary",
  onCancel,
  children,
}: {
  attempt: Attempt;
  look?: ButtonLook;
  onCancel: () => void;
  children: ReactNode;
}) {
  return (
    <>
      {attempt.failure}
      <DialogActions>
        <Button disabled={attempt.busy} onClick={onCancel}>
          Cancel
        </Button>
        <Button look={look} busy={attempt.busy} onClick={attempt.run}>
          {children}
        </Button>
      </DialogActions>
    </>
  );
}
