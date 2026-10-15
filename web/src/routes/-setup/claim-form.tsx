import { z } from "zod";

import { isProblem } from "../../api/errors";
import type { IdentityClaimed } from "../../api/gen/model";
import { claimController } from "../../api/gen/query/identity/identity";
import { ClaimControllerBody } from "../../api/gen/zod/identity/identity.zod";
import { formatSetupCode, normalizeSetupCode } from "../../session/setup-code";
import { submitErrors, useAppForm } from "../../ui/form";

const setupForm = ClaimControllerBody.extend({
  setupCode: z
    .string()
    .transform(normalizeSetupCode)
    .pipe(z.string().min(1, "Enter the Setup code")),
  pinAgain: z.string(),
}).refine((value) => value.pin === value.pinAgain, {
  message: "The PINs don't match",
  path: ["pinAgain"],
});

const displaySetupCode = (typed: string) => formatSetupCode(normalizeSetupCode(typed));

type Outcome = { onClaimed: (claimed: IdentityClaimed) => void; onAlreadyClaimed: () => void };

type Props = Outcome & { linkedSetupCode: string | undefined };

function useClaimForm({ onClaimed, onAlreadyClaimed, linkedSetupCode }: Props) {
  return useAppForm({
    defaultValues: {
      setupCode: displaySetupCode(linkedSetupCode ?? ""),
      email: "",
      displayName: "",
      pin: "",
      pinAgain: "",
    },
    validators: { onSubmit: setupForm },
    onSubmit: async ({ value, formApi }) => {
      const { setupCode, pinAgain: _pinAgain, ...body } = setupForm.parse(value);
      const claimed = await claimController(body, { headers: { "X-Setup-Code": setupCode } }).catch(
        (error: unknown) => {
          if (isProblem(error, "already_claimed")) {
            onAlreadyClaimed();
          } else {
            formApi.setErrorMap({
              onSubmit: submitErrors(error, { setup_code_wrong: "setupCode" }),
            });
          }
          return null;
        },
      );
      if (claimed !== null) {
        onClaimed(claimed);
      }
    },
  });
}

export function ClaimForm(props: Props) {
  const form = useClaimForm(props);
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
        <form.AppField name="setupCode">
          {(field) => (
            <field.TextField
              label="Setup code"
              autoCapitalize="characters"
              autoComplete="off"
              spellCheck={false}
              format={displaySetupCode}
              description={props.linkedSetupCode === undefined ? undefined : "Setup code filled in"}
            />
          )}
        </form.AppField>
        <form.AppField name="email">
          {(field) => <field.TextField label="Email" type="email" autoComplete="username" />}
        </form.AppField>
        <form.AppField name="displayName">
          {(field) => <field.TextField label="Display name" autoComplete="name" />}
        </form.AppField>
        <form.AppField name="pin">
          {(field) => <field.PinField label="PIN" autoComplete="new-password" />}
        </form.AppField>
        <form.AppField name="pinAgain">
          {(field) => <field.PinField label="PIN again" autoComplete="new-password" />}
        </form.AppField>
        <form.FormAlert />
        <form.SubmitButton label="Create Admin" />
      </form.AppForm>
    </form>
  );
}
