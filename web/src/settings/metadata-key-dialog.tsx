import { z } from "zod";

import { DialogActions } from "../account/dialog-actions";
import type { MetadataNewKey, MetadataProvider } from "../api/gen/model";
import { setMetadataKey } from "../api/gen/query/metadata/metadata";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { submitErrors, useAppForm } from "../ui/form";
import { toast } from "../ui/toast";

const keyForm = z.object({
  key: z.string().trim().min(1, "Enter the key.").max(256),
  pin: z.string().trim().max(256),
});

const bodyOf = ({ key, pin }: z.infer<typeof keyForm>): MetadataNewKey =>
  pin === "" ? { key } : { key, pin };

const refusedFields = { metadata_key_refused: "key", metadata_pin_refused: "pin" } as const;

const useKeyForm = (provider: MetadataProvider, onDone: () => void) =>
  useAppForm({
    defaultValues: { key: "", pin: "" },
    validators: { onSubmit: keyForm },
    onSubmit: async ({ value, formApi }) => {
      const body = bodyOf(keyForm.parse(value));
      try {
        await setMetadataKey(provider.id, body);
      } catch (error) {
        formApi.setErrorMap({ onSubmit: submitErrors(error, refusedFields) });
        return;
      }
      toast(
        `SlipStream now calls ${provider.label} with your ${body.pin === undefined ? "key" : "key and PIN"}.`,
      );
      onDone();
    },
  });

const intro = ({ label, takesPin }: MetadataProvider) =>
  takesPin
    ? `SlipStream makes one call to ${label} with the key and any PIN, and saves them only if ${label} accepts them.`
    : `SlipStream makes one call to ${label} with the key and saves it only if ${label} accepts it.`;

function KeyForm({ provider, onDone }: { provider: MetadataProvider; onDone: () => void }) {
  const form = useKeyForm(provider, onDone);
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
        <p className="text-ink-2">{intro(provider)}</p>
        <form.AppField name="key">
          {(field) => <field.TextField label="Key" data autoComplete="off" spellCheck={false} />}
        </form.AppField>
        {provider.takesPin && (
          <form.AppField name="pin">
            {(field) => (
              <field.TextField
                label="PIN"
                description="Leave it empty unless your key comes with a subscriber PIN."
                data
                autoComplete="off"
                spellCheck={false}
              />
            )}
          </form.AppField>
        )}
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
