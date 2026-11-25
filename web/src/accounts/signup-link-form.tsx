import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { z } from "zod";

import { isProblem } from "../api/errors";
import type {
  IdentityAccountRow,
  IdentityIssuedSignupLink,
  IdentityServerGrant,
  MediaserversServerView,
} from "../api/gen/model";
import {
  createSignupLink,
  getAddress,
  setAddress,
  useListAccountsSuspense,
} from "../api/gen/query/identity/identity";
import { useListMediaServersSuspense } from "../api/gen/query/mediaservers/mediaservers";
import { submitErrors, useAppForm } from "../ui/form";
import { MediaServerChecklist, shownGrants } from "./media-server-checklist";

const linkForm = z.object({
  email: z.string().trim().pipe(z.email("Enter an email address")),
  address: z.string().trim(),
  mediaServers: z.array(z.custom<IdentityServerGrant>()),
});

function useSignupLinkForm(
  servers: readonly MediaserversServerView[],
  onIssued: (issued: IdentityIssuedSignupLink) => void,
) {
  const [needsAddress, setNeedsAddress] = useState(false);
  const [existing, setExisting] = useState<IdentityAccountRow>();
  const { accounts } = useListAccountsSuspense().data;
  const form = useAppForm({
    defaultValues: { email: "", address: "", mediaServers: shownGrants(servers) },
    validators: { onSubmit: linkForm },
    onSubmit: async ({ value, formApi }) => {
      const { email, address, mediaServers } = linkForm.parse(value);
      setExisting(undefined);
      try {
        if (needsAddress) {
          await setAddress({ address });
          setNeedsAddress(false);
        }
        onIssued(await createSignupLink({ email, mediaServers }));
      } catch (error) {
        if (isProblem(error, "address_unset")) {
          const { suggestedAddress } = await getAddress();
          formApi.setFieldValue("address", suggestedAddress);
          setNeedsAddress(true);
        }
        if (isProblem(error, "account_exists")) {
          const lower = email.toLowerCase();
          setExisting(accounts.find((account) => account.email.toLowerCase() === lower));
        }
        formApi.setErrorMap({
          onSubmit: submitErrors(error, { account_exists: "email", address_ip: "address" }),
        });
      }
    },
  });
  return { form, needsAddress, existing };
}

function ExistingAccount({ account }: { account: IdentityAccountRow }) {
  return (
    <p>
      Offer Media Servers from{" "}
      <Link
        to="/accounts/$accountId"
        params={{ accountId: account.id }}
        className="text-ink-1 underline"
      >
        their Account page
      </Link>
      .
    </p>
  );
}

export function LinkForm({ onIssued }: { onIssued: (issued: IdentityIssuedSignupLink) => void }) {
  const { servers } = useListMediaServersSuspense().data;
  const { form, needsAddress, existing } = useSignupLinkForm(servers, onIssued);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <p className="text-ink-2">
        The person opens the link to make their own Member Account with a name and a PIN.
      </p>
      <form.AppForm>
        <form.AppField name="email">
          {(field) => <field.TextField label="Email" type="email" autoComplete="off" />}
        </form.AppField>
        {existing !== undefined && <ExistingAccount account={existing} />}
        {needsAddress ? (
          <form.AppField name="address">
            {(field) => (
              <field.TextField
                label="SlipStream's address"
                type="url"
                autoComplete="off"
                spellCheck={false}
                description="Links use this address. It's saved for every link after this one."
              />
            )}
          </form.AppField>
        ) : null}
        <form.Field name="mediaServers">
          {(field) => (
            <MediaServerChecklist
              servers={servers}
              value={field.state.value}
              onChange={field.handleChange}
            />
          )}
        </form.Field>
        <form.FormAlert />
        <div className="flex justify-end">
          <form.SubmitButton label="Make the link" />
        </div>
      </form.AppForm>
    </form>
  );
}
