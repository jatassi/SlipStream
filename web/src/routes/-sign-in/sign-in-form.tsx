import { FingerPrintIcon } from "@hugeicons/core-free-icons";
import { z } from "zod";

import type { IdentitySignedIn } from "../../api/gen/model";
import { useCreateSession } from "../../api/gen/query/identity/identity";
import { CreateSessionBody } from "../../api/gen/zod/identity/identity.zod";
import { readPref } from "../../shell/prefs";
import { Button } from "../../ui/button";
import { useAppForm } from "../../ui/form";
import { usePasskeySignIn } from "./passkey-sign-in";
import { signInErrors } from "./signed-in";

const signInForm = CreateSessionBody.extend({
  email: z.string().trim().pipe(CreateSessionBody.shape.email),
});

type SignedIn = (session: IdentitySignedIn) => Promise<void>;

function useSignInForm(signedIn: SignedIn) {
  const createSession = useCreateSession();
  return useAppForm({
    defaultValues: { email: readPref("lastEmail"), pin: "" },
    validators: { onSubmit: signInForm },
    onSubmit: async ({ value, formApi }) => {
      const session = await createSession
        .mutateAsync({ data: signInForm.parse(value) })
        .catch((error: unknown) => {
          formApi.setFieldValue("pin", "");
          formApi.setErrorMap({ onSubmit: signInErrors(error) });
          return null;
        });
      if (session !== null) {
        await signedIn(session);
      }
    },
  });
}

export function SignInForm({ signedIn }: { signedIn: SignedIn }) {
  const form = useSignInForm(signedIn);
  const passkey = usePasskeySignIn({
    signedIn,
    failed: (error) => {
      form.setErrorMap({ onSubmit: signInErrors(error) });
    },
  });
  return (
    <>
      <form
        noValidate
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.AppForm>
          <form.AppField name="email">
            {(field) => (
              <field.TextField label="Email" type="email" autoComplete="username webauthn" />
            )}
          </form.AppField>
          <form.AppField name="pin">
            {(field) => <field.PinPad label="PIN" autoComplete="current-password" autoSubmit />}
          </form.AppField>
          <form.FormAlert />
          <form.SubmitButton label="Sign in" />
        </form.AppForm>
      </form>
      {passkey.shown && (
        <Button
          icon={FingerPrintIcon}
          busy={passkey.busy}
          onClick={() => void passkey.withButton()}
        >
          Sign in with a passkey
        </Button>
      )}
    </>
  );
}
