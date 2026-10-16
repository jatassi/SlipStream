import { useRouter } from "@tanstack/react-router";
import { z } from "zod";

import {
  updateIdentitySettings,
  useGetIdentitySettingsSuspense,
} from "../api/gen/query/identity/identity";
import { submitErrors, useAppForm } from "../ui/form";
import { toast } from "../ui/toast";

const addressForm = z.object({ address: z.string().trim().min(1, "Enter an address") });

function describe(address: string, saved: string): string {
  if (address.trim().startsWith("http://")) {
    return "A plain-HTTP address can't hold passkeys. Use https:// for passkeys to work.";
  }
  return saved === ""
    ? "Not set yet. Signup, Reset and email links can't go out until it is."
    : "Signup, Reset and email links are built on this address.";
}

export function AddressSection() {
  const { address, suggestedAddress } = useGetIdentitySettingsSuspense().data;
  const router = useRouter();
  const form = useAppForm({
    defaultValues: { address: address === "" ? suggestedAddress : address },
    validators: { onSubmit: addressForm },
    onSubmit: async ({ value, formApi }) => {
      const saved = await updateIdentitySettings(addressForm.parse(value)).catch(
        (error: unknown) => {
          formApi.setErrorMap({ onSubmit: submitErrors(error, { address_ip: "address" }) });
          return null;
        },
      );
      if (saved !== null) {
        formApi.reset({ address: saved.address });
        toast("Saved the address");
        await router.invalidate();
      }
    },
  });
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
        <form.AppField name="address">
          {(field) => (
            <field.TextField
              label="Address"
              type="url"
              autoComplete="off"
              spellCheck={false}
              description={describe(field.state.value, address)}
            />
          )}
        </form.AppField>
        <form.FormAlert />
        <div>
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}
