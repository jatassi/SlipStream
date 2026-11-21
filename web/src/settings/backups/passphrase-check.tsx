import { Tick02Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import { checkBackupPassphrase } from "../../api/gen/query/backups/backups";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { useAppForm } from "../../ui/form";
import { Icon } from "../../ui/icon";
import { SignalLine } from "../../ui/signal-line";

type Checked = "matches" | "differs";

type Answer = { readonly checked?: Checked; readonly refusal?: string };

function Answered({ answer }: { answer: Answer }) {
  return (
    <>
      {answer.checked === "matches" && (
        <output className="flex items-center gap-1.5 text-ink-1">
          <Icon icon={Tick02Icon} className="size-4 shrink-0 text-ok" />
          That&apos;s your current passphrase.
        </output>
      )}
      {answer.checked === "differs" && (
        <SignalLine tone="fail" announce>
          That isn&apos;t your current passphrase.
        </SignalLine>
      )}
      {answer.refusal !== undefined && (
        <SignalLine tone="fail" announce>
          {answer.refusal}
        </SignalLine>
      )}
    </>
  );
}

function CheckForm({ close }: { close: () => void }) {
  const [answer, setAnswer] = useState<Answer>({});
  const form = useAppForm({
    defaultValues: { passphrase: "" },
    onSubmit: async ({ value }) => {
      try {
        const { matches } = await checkBackupPassphrase({ passphrase: value.passphrase.trim() });
        setAnswer({ checked: matches ? "matches" : "differs" });
      } catch (error) {
        setAnswer({ refusal: sentenceOf(error) });
      }
    },
  });
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <form.AppField name="passphrase">
          {(field) => (
            <field.TextField
              label="Saved passphrase"
              autoComplete="off"
              autoCapitalize="off"
              spellCheck={false}
            />
          )}
        </form.AppField>
        <Answered answer={answer} />
        <div className="flex justify-end gap-2">
          <Button onClick={close}>Close</Button>
          <form.SubmitButton label="Check" />
        </div>
      </form.AppForm>
    </form>
  );
}

export function CheckSavedCopy() {
  const [open, setOpen] = useState(false);
  return (
    <Dialog
      title="Check my saved copy"
      open={open}
      onOpenChange={setOpen}
      trigger={<Button>Check my saved copy</Button>}
    >
      {open && (
        <CheckForm
          close={() => {
            setOpen(false);
          }}
        />
      )}
    </Dialog>
  );
}
