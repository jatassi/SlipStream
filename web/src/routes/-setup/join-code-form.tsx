import { useNavigate } from "@tanstack/react-router";
import { z } from "zod";

import { isApiError } from "../../api/errors";
import { claimWorker } from "../../api/gen/query/pairing/pairing";
import { forgetSetupCode, setupCodeHeaders } from "../../session/setup-code";
import { Button } from "../../ui/button";
import { submitErrors, useAppForm } from "../../ui/form";
import type { FormProps } from "./form-props";
import { type JoinRefusal, isJoinRefusal, joinSentences } from "./join-sentences";

const joinForm = z.object({
  joinCode: z.string().trim().min(1, joinSentences.invalid.text),
});

type Outcomes = {
  readonly waiting: () => Promise<unknown>;
  readonly refused: (refusal: JoinRefusal) => void;
};

function useJoinForm({ code, restarted }: FormProps, { waiting, refused }: Outcomes) {
  const navigate = useNavigate();
  return useAppForm({
    defaultValues: { joinCode: "" },
    validators: { onSubmit: joinForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        const claim = await claimWorker(
          { code: joinForm.parse(value).joinCode },
          { headers: setupCodeHeaders(code) },
        );
        if (claim.state === "joined") {
          forgetSetupCode();
          await navigate({ to: "/worker" });
        } else {
          await waiting();
        }
      } catch (error) {
        if (!isApiError(error) || !isJoinRefusal(error.code)) {
          formApi.setErrorMap({ onSubmit: submitErrors(error) });
          return;
        }
        const { text, next } = joinSentences[error.code];
        if (next === "code") {
          restarted();
        } else if (next === "retry") {
          formApi.setErrorMap({ onSubmit: { form: text, fields: {} } });
        } else {
          refused(error.code);
        }
      }
    },
  });
}

export function JoinCodeForm({ props, outcomes }: { props: FormProps; outcomes: Outcomes }) {
  const form = useJoinForm(props, outcomes);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <p>Paste the Join Code from Add a Node on your SlipStream.</p>
      <form.AppForm>
        <form.AppField name="joinCode">
          {(field) => (
            <field.TextField
              label="Join Code"
              data
              autoCapitalize="characters"
              autoComplete="off"
              spellCheck={false}
            />
          )}
        </form.AppField>
        <form.FormAlert />
        <div className="flex flex-wrap items-center gap-3">
          <form.SubmitButton label="Join" />
          <Button look="ghost" onClick={props.back}>
            Back
          </Button>
        </div>
      </form.AppForm>
    </form>
  );
}
