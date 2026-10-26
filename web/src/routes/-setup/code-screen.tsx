import { useNavigate } from "@tanstack/react-router";
import { z } from "zod";

import { isProblem } from "../../api/errors";
import {
  type SetupCode,
  checkedSetupCode,
  formatSetupCode,
  normalizeSetupCode,
} from "../../session/setup-code";
import { AlertLine } from "../../ui/alert-line";
import { submitErrors, useAppForm } from "../../ui/form";
import { Spinner } from "../../ui/spinner";
import { busySentence } from "./join-sentences";

const codeForm = z.object({
  setupCode: z
    .string()
    .transform(normalizeSetupCode)
    .pipe(z.string().min(1, "Enter the Setup code")),
});

const displaySetupCode = (typed: string) => formatSetupCode(normalizeSetupCode(typed));

export function CheckingCode() {
  return (
    <p className="flex items-center gap-2">
      <Spinner />
      Checking the Setup code…
    </p>
  );
}

function useCodeForm(accept: (code: SetupCode) => void) {
  const navigate = useNavigate();
  return useAppForm({
    defaultValues: { setupCode: "" },
    validators: { onSubmit: codeForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        accept(await checkedSetupCode(codeForm.parse(value).setupCode));
      } catch (error) {
        if (isProblem(error, "already_claimed")) {
          await navigate({ to: "/sign-in" });
          return;
        }
        formApi.setErrorMap({
          onSubmit: isProblem(error, "setup_busy")
            ? { form: busySentence, fields: {} }
            : submitErrors(error, { setup_code_wrong: "setupCode" }),
        });
      }
    },
  });
}

/** The first screen: the Setup code alone, checked before anything else is asked. */
export function CodeScreen({
  note,
  accept,
}: {
  note: string | undefined;
  accept: (code: SetupCode) => void;
}) {
  const form = useCodeForm(accept);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      {note === undefined ? null : <AlertLine text={note} />}
      <p>Enter the Setup code from this Node&apos;s console or log.</p>
      <form.AppForm>
        <form.AppField name="setupCode">
          {(field) => (
            <field.TextField
              label="Setup code"
              autoCapitalize="characters"
              autoComplete="off"
              spellCheck={false}
              format={displaySetupCode}
            />
          )}
        </form.AppField>
        <form.FormAlert />
        <form.SubmitButton label="Continue" />
      </form.AppForm>
    </form>
  );
}
