import { useState } from "react";

import { isApiError, sentenceOf } from "../api/errors";
import type { MediaserversKindView as Kind } from "../api/gen/model";
import { signInToMediaServer } from "../api/gen/query/mediaservers/mediaservers";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import { useAppForm } from "../ui/form";
import { submitErrors } from "../ui/form-context";
import { SignalLine } from "./lines";
import type { ServerForm } from "./use-server-form";

/**
 * Signs in once as the server's administrator, from the Node the draft names,
 * and fills the key field with the key the sign-in created. The password goes
 * in that one call and stays out of the draft.
 */
function useAdminSignIn(form: ServerForm, kind: Kind) {
  const [created, setCreated] = useState(false);
  const signIn = useAppForm({
    defaultValues: { user: "", password: "" },
    onSubmit: async ({ value, formApi }) => {
      const draft = form.state.values;
      try {
        const got = await signInToMediaServer({
          kind: kind.kind,
          address: draft.address.trim(),
          nodeId: draft.reachingNodeId,
          user: value.user.trim(),
          password: value.password,
        });
        form.setFieldValue("apiKey", got.apiKey);
        form.setFieldValue("deviceId", got.deviceId);
        formApi.setFieldValue("password", "");
        setCreated(true);
      } catch (error) {
        formApi.setErrorMap({
          onSubmit:
            isApiError(error) && error.code !== "invalid"
              ? { form: sentenceOf(error), fields: {} }
              : submitErrors(error),
        });
      }
    },
  });
  return { signIn, created };
}

export function AdminSignIn({ form, kind }: { form: ServerForm; kind: Kind }) {
  const { signIn, created } = useAdminSignIn(form, kind);
  return (
    <Details fields label="Sign in as an administrator">
      <div className="mt-2 flex flex-col gap-4">
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          {`SlipStream creates its own key named SlipStream on ${kind.label} and doesn't keep the password.`}
        </p>
        <signIn.AppForm>
          <signIn.AppField name="user">
            {(field) => <field.TextField label="User" autoComplete="off" spellCheck={false} />}
          </signIn.AppField>
          <signIn.AppField name="password">
            {(field) => <field.SecretField label="Password" saved={false} />}
          </signIn.AppField>
          <signIn.FormAlert />
          <div>
            <signIn.Subscribe selector={(state) => state.isSubmitting}>
              {(busy) => (
                <Button busy={busy} onClick={() => void signIn.handleSubmit()}>
                  Sign in
                </Button>
              )}
            </signIn.Subscribe>
          </div>
        </signIn.AppForm>
        {created && <SignalLine level="ok" text="Created a key named SlipStream." />}
      </div>
    </Details>
  );
}
