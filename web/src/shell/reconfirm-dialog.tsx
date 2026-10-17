import { useEffect, useState } from "react";

import { useReconfirmSession } from "../api/gen/query/identity/identity";
import { ReconfirmSessionBody } from "../api/gen/zod/identity/identity.zod";
import { settleReconfirm, useReconfirmAsk } from "../api/reconfirm";
import { pinErrors } from "../session/pin-errors";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { useAppForm } from "../ui/form";
import { useShell } from "../ui/shell-context";

export type ReconfirmDialogProps = {
  /**
   * Confirms with a passkey instead, shown as "Use a passkey" when given. It
   * resolves true once the Node confirmed, and false when the person backed out.
   */
  passkey?: (() => Promise<boolean>) | undefined;
};

function useReconfirmForm() {
  const reconfirm = useReconfirmSession();
  return useAppForm({
    defaultValues: { pin: "" },
    validators: { onSubmit: ReconfirmSessionBody },
    onSubmit: async ({ value, formApi }) => {
      const confirmed = await reconfirm.mutateAsync({ data: value }).then(
        () => true,
        (error: unknown) => {
          formApi.setFieldValue("pin", "");
          formApi.setErrorMap({ onSubmit: pinErrors(error, "That PIN is wrong.") });
          return false;
        },
      );
      if (confirmed) {
        settleReconfirm(true);
      }
    },
  });
}

function usePasskeyConfirm(passkey: () => Promise<boolean>) {
  const [busy, setBusy] = useState(false);
  const [refusal, setRefusal] = useState<string>();
  const confirm = async () => {
    setBusy(true);
    setRefusal(undefined);
    try {
      if (await passkey()) {
        settleReconfirm(true);
      }
    } catch (error) {
      setRefusal(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy(false);
    }
  };
  return { busy, refusal, confirm };
}

const noPasskey = () => Promise.resolve(false);

function ReconfirmForm({ passkey }: ReconfirmDialogProps) {
  const form = useReconfirmForm();
  const stacked = useShell() === "phone";
  const passkeyConfirm = usePasskeyConfirm(passkey ?? noPasskey);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <p className="text-ink-2">Enter your PIN to continue.</p>
      <form.AppForm>
        <form.AppField name="pin">
          {(field) => <field.PinPad label="PIN" autoComplete="current-password" autoSubmit />}
        </form.AppField>
        <form.FormAlert />
        {passkeyConfirm.refusal !== undefined && <AlertLine text={passkeyConfirm.refusal} />}
        <div className={`flex gap-2 ${stacked ? "flex-col-reverse" : "justify-end"}`}>
          {passkey && (
            <Button
              look="ghost"
              busy={passkeyConfirm.busy}
              onClick={() => void passkeyConfirm.confirm()}
            >
              Use a passkey
            </Button>
          )}
          <Button
            onClick={() => {
              settleReconfirm(false);
            }}
          >
            Cancel
          </Button>
          <form.SubmitButton label="Confirm" />
        </div>
      </form.AppForm>
    </form>
  );
}

/**
 * The one re-confirm dialog. The API client opens it when an action answers
 * needs_reconfirm, and each signed-in frame mounts it, so a question left open
 * is cancelled with the session that asked it.
 */
export function ReconfirmDialog({ passkey }: ReconfirmDialogProps) {
  const ask = useReconfirmAsk();
  const [shown, setShown] = useState(ask ?? 0);
  if (ask !== undefined && ask !== shown) {
    setShown(ask);
  }
  useEffect(
    () => () => {
      settleReconfirm(false);
    },
    [],
  );
  return (
    <Dialog
      title="Confirm it's you"
      open={ask !== undefined}
      onOpenChange={(open) => {
        if (!open) {
          settleReconfirm(false);
        }
      }}
    >
      <ReconfirmForm key={shown} passkey={passkey} />
    </Dialog>
  );
}
