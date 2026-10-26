import { z } from "zod";

import { DialogActions } from "../account/dialog-actions";
import type { MetadataProvider } from "../api/gen/model";
import { setMetadataKey } from "../api/gen/query/metadata/metadata";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { submitErrors, useAppForm } from "../ui/form";
import { toast } from "../ui/toast";

const keyForm = z.object({ key: z.string().trim().min(1, "Enter the key.").max(256) });

function KeyForm({ provider, onDone }: { provider: MetadataProvider; onDone: () => void }) {
  const form = useAppForm({
    defaultValues: { key: "" },
    validators: { onSubmit: keyForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        await setMetadataKey(provider.id, keyForm.parse(value));
      } catch (error) {
        formApi.setErrorMap({ onSubmit: submitErrors(error, { metadata_key_refused: "key" }) });
        return;
      }
      toast(`${provider.label} now uses your key.`);
      onDone();
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
        <p className="text-ink-2">
          SlipStream makes one call to {provider.label} with the key and saves it only if{" "}
          {provider.label} accepts it.
        </p>
        <form.AppField name="key">
          {(field) => <field.TextField label="Key" data autoComplete="off" spellCheck={false} />}
        </form.AppField>
        <form.FormAlert />
        <DialogActions>
          <DialogClose>
            <Button>Cancel</Button>
          </DialogClose>
          <form.SubmitButton label="Test and save" />
        </DialogActions>
      </form.AppForm>
    </form>
  );
}

/** Takes the Admin's key for a provider, which the provider must accept on a test call before it saves. */
export function KeyDialog({
  provider,
  open,
  onOpenChange,
}: {
  provider: MetadataProvider;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title={`Set your ${provider.label} key`} open={open} onOpenChange={onOpenChange}>
      <KeyForm
        provider={provider}
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}
