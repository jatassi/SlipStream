import { useState } from "react";
import { z } from "zod";

import { dialForCode } from "../../../../api/gen/query/nodes/nodes";
import { Button } from "../../../../ui/button";
import { submitErrors, useAppForm } from "../../../../ui/form";
import { useRefreshCodes } from "./codes";
import { DialLine } from "./dial-line";
import type { Dial } from "./state";

const addressForm = z.object({ nodeAddress: z.string().trim().min(1, "Paste the Node address") });

function AddressForm({ id, controller }: { id: string; controller: string }) {
  const refreshCodes = useRefreshCodes();
  const form = useAppForm({
    defaultValues: { nodeAddress: "" },
    validators: { onSubmit: addressForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        await dialForCode(id, addressForm.parse(value));
        await refreshCodes();
      } catch (error) {
        formApi.setErrorMap({
          onSubmit: submitErrors(error, { node_address_damaged: "nodeAddress" }),
        });
      }
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
        <form.AppField name="nodeAddress">
          {(field) => (
            <field.TextField
              label="Node address"
              autoComplete="off"
              spellCheck={false}
              description={`The new Node prints it when it can't reach ${controller}. It starts with SLN1.`}
            />
          )}
        </form.AppField>
        <form.FormAlert />
        <div className="flex justify-end">
          <form.SubmitButton label={`${controller} dials it`} />
        </div>
      </form.AppForm>
    </form>
  );
}

/**
 * For a new Node that can't reach the Controller: paste the Node address it
 * printed, and the Controller dials it instead.
 */
export function NodeAddress({
  id,
  controller,
  dial,
}: {
  id: string;
  controller: string;
  dial: Dial;
}) {
  const [asked, setAsked] = useState(false);
  if (!asked && dial.kind === "none") {
    return (
      <div>
        <Button
          look="ghost"
          onClick={() => {
            setAsked(true);
          }}
        >
          {`The new Node can't reach ${controller}?`}
        </Button>
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-3">
      <DialLine dial={dial} controller={controller} />
      <AddressForm id={id} controller={controller} />
    </div>
  );
}
