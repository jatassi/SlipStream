import type { ReactNode } from "react";
import { z } from "zod";

import { submitErrors, useAppForm } from "../ui/form";

const publicForm = z.object({
  public: z.string().trim().min(1, "Enter the address players outside your home use"),
});

const fieldOf = { public_required: "public", public_not_url: "public" } as const;

export function PublicForm({
  initial,
  save,
  actions,
}: {
  initial: string;
  save: (address: string) => Promise<unknown>;
  actions: ReactNode;
}) {
  const form = useAppForm({
    defaultValues: { public: initial },
    validators: { onSubmit: publicForm },
    onSubmit: async ({ value, formApi }) => {
      await save(publicForm.parse(value).public).catch((error: unknown) => {
        formApi.setErrorMap({ onSubmit: submitErrors(error, fieldOf) });
      });
    },
  });
  return (
    <form
      noValidate
      className="flex flex-col gap-3"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <form.AppField name="public">
          {(field) => (
            <field.TextField
              label="Public address"
              type="url"
              autoComplete="off"
              spellCheck={false}
              data
              description="The address players outside your home use."
            />
          )}
        </form.AppField>
        <form.FormAlert />
        {actions}
      </form.AppForm>
    </form>
  );
}
