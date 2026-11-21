import { useState } from "react";
import { z } from "zod";

import { isApiError } from "../../api/errors";
import type { PairingWorkerClaim } from "../../api/gen/model";
import { claimWorker } from "../../api/gen/query/pairing/pairing";
import { normalizeSetupCode } from "../../session/setup-code";
import { submitErrors, useAppForm } from "../../ui/form";
import { isJoinRefusal, joinSentences } from "./join-sentences";
import { JoinWait } from "./join-wait";

const pairForm = z.object({
  setupCode: z.string().trim().min(1, "Enter the Setup code from this Node's log."),
  joinCode: z.string().trim().min(1, joinSentences.invalid.text),
});

const notOpen =
  "This Node takes a Join Code here only once its log shows a Setup code, after it has been unable to reach its Controller for a while.";

function usePairForm(pairing: (claim: PairingWorkerClaim) => void) {
  return useAppForm({
    defaultValues: { setupCode: "", joinCode: "" },
    validators: { onSubmit: pairForm },
    onSubmit: async ({ value, formApi }) => {
      const { setupCode, joinCode } = pairForm.parse(value);
      try {
        pairing(
          await claimWorker(
            { code: joinCode },
            { headers: { "X-Setup-Code": normalizeSetupCode(setupCode) } },
          ),
        );
      } catch (error) {
        if (isApiError(error) && error.code === "setup_code_wrong") {
          formApi.setErrorMap({
            onSubmit: { fields: { setupCode: "That isn't this Node's newest Setup code." } },
          });
        } else if (isApiError(error) && error.code === "already_claimed") {
          formApi.setErrorMap({ onSubmit: { form: notOpen, fields: {} } });
        } else if (isApiError(error) && isJoinRefusal(error.code)) {
          formApi.setErrorMap({ onSubmit: { form: joinSentences[error.code].text, fields: {} } });
        } else {
          formApi.setErrorMap({ onSubmit: submitErrors(error) });
        }
      }
    },
  });
}

/** A paired Worker that can't reach its Controller pairs here with another one. */
export function PairAgain({ name }: { name: string }) {
  const [claim, setClaim] = useState<PairingWorkerClaim>();
  const form = usePairForm(setClaim);
  if (claim?.state === "awaiting-dial") {
    return <JoinWait claim={claim} />;
  }
  if (claim !== undefined) {
    return <p>{`${name} joined the new Controller and restarts now.`}</p>;
  }
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <p>{`To pair ${name} with another Controller, enter the Setup code from its log and a Join Code from that Controller.`}</p>
      <form.AppForm>
        <form.AppField name="setupCode">
          {(field) => (
            <field.TextField label="Setup code" data autoComplete="off" spellCheck={false} />
          )}
        </form.AppField>
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
        <div>
          <form.SubmitButton label="Pair again" />
        </div>
      </form.AppForm>
    </form>
  );
}
