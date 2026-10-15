import { createFormHook } from "@tanstack/react-form";

import { Button } from "./button";
import { FormAlert } from "./form-alert";
import { fieldContext, formContext, useFormContext } from "./form-context";
import { PinField } from "./pin-field";
import { TextField } from "./text-field";

function SubmitButton({ label }: { label: string }) {
  const form = useFormContext();
  return (
    <form.Subscribe selector={(state) => state.isSubmitting}>
      {(submitting) => (
        <Button type="submit" look="primary" busy={submitting}>
          {label}
        </Button>
      )}
    </form.Subscribe>
  );
}

export { submitErrors } from "./form-context";

export const { useAppForm } = createFormHook({
  fieldContext,
  formContext,
  fieldComponents: { TextField, PinField },
  formComponents: { SubmitButton, FormAlert },
});
