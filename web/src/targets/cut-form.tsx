import { z } from "zod";

import { DialogActions } from "../account/dialog-actions";
import { isProblem } from "../api/errors";
import type { TargetsNewCut } from "../api/gen/model";
import { Button } from "../ui/button";
import { DialogClose } from "../ui/dialog";
import { submitErrors, useAppForm } from "../ui/form";
import { type CutDraft, cutBodyOf } from "./cuts";

const cutForm = z.object({
  name: z.string().trim().min(1, "Name the cut.").max(100, "Keep the name to 100 characters."),
  aliases: z.string(),
});

const nameTaken = "Another cut already uses this name or one of these phrases.";

function refusedWith(error: unknown) {
  return isProblem(error, "cut_name_taken") ? { fields: { name: nameTaken } } : submitErrors(error);
}

function useCutForm(initial: CutDraft, onSubmit: (body: TargetsNewCut) => Promise<void>) {
  return useAppForm({
    defaultValues: initial,
    validators: { onSubmit: cutForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        await onSubmit(cutBodyOf(value));
      } catch (error) {
        formApi.setErrorMap({ onSubmit: refusedWith(error) });
      }
    },
  });
}

export function CutForm({
  initial,
  submitLabel,
  onSubmit,
}: {
  initial: CutDraft;
  submitLabel: string;
  onSubmit: (body: TargetsNewCut) => Promise<void>;
}) {
  const form = useCutForm(initial, onSubmit);
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
        <form.AppField name="name">
          {(field) => <field.TextField label="Name" autoComplete="off" />}
        </form.AppField>
        <form.AppField name="aliases">
          {(field) => (
            <field.TextField
              label="Also read as"
              autoComplete="off"
              description="Other phrases a title names this cut by, split by commas, such as B&C, Black and Chrome. Its name always counts."
            />
          )}
        </form.AppField>
        <form.FormAlert />
        <DialogActions>
          <DialogClose>
            <Button>Cancel</Button>
          </DialogClose>
          <form.SubmitButton label={submitLabel} />
        </DialogActions>
      </form.AppForm>
    </form>
  );
}
