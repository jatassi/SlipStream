import { useRouter } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { z } from "zod";

import { setAddress, useGetAddressSuspense } from "../api/gen/query/identity/identity";
import { SubmitButton, submitErrors, useAppForm } from "../ui/form";
import { toast } from "../ui/toast";

const addressForm = z.object({ address: z.string().trim().min(1, "Enter an address") });

// Browsers treat localhost and *.localhost as secure, so plain HTTP holds a
// passkey there, as the relying-party rule on the server says.
function refusesPasskeys(address: string): boolean {
  const url = URL.parse(address.trim());
  return (
    url?.protocol === "http:" &&
    url.hostname !== "localhost" &&
    !url.hostname.endsWith(".localhost")
  );
}

function describe(address: string, saved: string): string {
  if (refusesPasskeys(address)) {
    return "Passkeys don't work on a plain-HTTP address. Use an https:// address.";
  }
  return saved === ""
    ? "Not set yet. SlipStream can't send Signup, Reset or email links until it is."
    : "Signup, Reset and email links use this address.";
}

type Saves = {
  onSaved: () => Promise<void>;
  /** Runs in place of a save when the field still holds the saved address. */
  onUnchanged?: (() => Promise<void>) | undefined;
};

function useAddressForm({ onSaved, onUnchanged }: Saves) {
  const { address, suggestedAddress } = useGetAddressSuspense().data;
  const form = useAppForm({
    defaultValues: { address: address === "" ? suggestedAddress : address },
    validators: { onSubmit: addressForm },
    onSubmit: async ({ value, formApi }) => {
      if (onUnchanged !== undefined && value.address === address) {
        await onUnchanged();
        return;
      }
      const saved = await setAddress(addressForm.parse(value)).catch((error: unknown) => {
        formApi.setErrorMap({ onSubmit: submitErrors(error, { address_ip: "address" }) });
        return null;
      });
      if (saved !== null) {
        formApi.reset({ address: saved.address });
        await onSaved();
      }
    },
  });
  return { form, address };
}

/** The address with the suggestion filled in while none is set; `children` hold its submit. */
export function AddressForm({ children, ...saves }: Saves & { children: ReactNode }) {
  const { form, address } = useAddressForm(saves);
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
        {children}
      </form.AppForm>
    </form>
  );
}

export function AddressSection() {
  const router = useRouter();
  return (
    <AddressForm
      onSaved={async () => {
        toast("Saved the address");
        await router.invalidate();
      }}
    >
      <div>
        <SubmitButton label="Save" />
      </div>
    </AddressForm>
  );
}
