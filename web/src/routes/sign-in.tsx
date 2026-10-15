import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { useCreateSession } from "../api/gen/query/identity/identity";
import { CreateSessionBody } from "../api/gen/zod/identity/identity.zod";
import type { SessionEndReason } from "../live/types";
import { requireEntry } from "../session/gate";
import { safeRedirect, signInSearch } from "../session/safe-redirect";
import { markSignedIn } from "../session/session";
import { EntryPage } from "../shell/entry-page";
import { submitErrors, useAppForm } from "../ui/form";

const signInForm = CreateSessionBody.extend({
  email: z.string().trim().pipe(CreateSessionBody.shape.email),
});

const reasonText: Record<SessionEndReason, string> = {
  "signed-out": "You signed out.",
  expired: "Your session ended. Sign in again.",
};

export const Route = createFileRoute("/sign-in")({
  validateSearch: signInSearch,
  beforeLoad: ({ context, location, search: { redirect } }) =>
    requireEntry(context, "sign-in", location, redirect),
  component: SignIn,
});

function useSignInForm() {
  const { redirect } = Route.useSearch();
  const { queryClient } = Route.useRouteContext();
  const navigate = Route.useNavigate();
  const createSession = useCreateSession();
  return useAppForm({
    defaultValues: { email: "", pin: "" },
    validators: { onSubmit: signInForm },
    onSubmit: async ({ value, formApi }) => {
      const session = await createSession
        .mutateAsync({ data: signInForm.parse(value) })
        .catch((error: unknown) => {
          formApi.setFieldValue("pin", "");
          formApi.setErrorMap({ onSubmit: submitErrors(error) });
          return null;
        });
      if (session !== null) {
        markSignedIn(queryClient, session);
        await navigate({ href: safeRedirect(redirect) });
      }
    },
  });
}

function SignIn() {
  const { reason } = Route.useSearch();
  const form = useSignInForm();
  return (
    <EntryPage title="Sign in to SlipStream">
      {reason === undefined ? null : <output className="text-ink-1">{reasonText[reason]}</output>}
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
            {(field) => <field.TextField label="Email" type="email" autoComplete="username" />}
          </form.AppField>
          <form.AppField name="pin">
            {(field) => <field.PinField label="PIN" autoComplete="current-password" autoSubmit />}
          </form.AppField>
          <form.FormAlert />
          <form.SubmitButton label="Sign in" />
        </form.AppForm>
      </form>
    </EntryPage>
  );
}
