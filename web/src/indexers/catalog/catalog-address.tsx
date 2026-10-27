import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { z } from "zod";

import { isProblem, sentenceOf } from "../../api/errors";
import type { IndexersIndexerCatalog } from "../../api/gen/model";
import { setCatalogAddress } from "../../api/gen/query/indexers/indexers";
import { adoptCatalog } from "../../live/adopt";
import { Button } from "../../ui/button";
import { submitErrors, useAppForm } from "../../ui/form";
import { toast } from "../../ui/toast";

const addressSchema = z.object({
  address: z
    .string()
    .trim()
    .min(1, "Enter the catalog's address")
    .max(2048, "Use 2,048 characters or fewer"),
});

/** The address serves no catalog: the title and what to check show under the field. */
const refusal = (error: unknown) =>
  isProblem(error, "catalog_address_unreachable")
    ? { fields: { address: `${error.title}. ${sentenceOf(error)}` } }
    : submitErrors(error);

function useSave() {
  const queryClient = useQueryClient();
  return async (address: string | null) => {
    adoptCatalog(queryClient, await setCatalogAddress({ address }));
  };
}

function UseDefault() {
  const save = useSave();
  const [busy, setBusy] = useState(false);
  const reset = async () => {
    setBusy(true);
    try {
      await save(null);
      toast("The Indexer catalog uses Prowlarr's address again");
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setBusy(false);
    }
  };
  return (
    <Button busy={busy} onClick={() => void reset()}>
      Use the default
    </Button>
  );
}

/** Where the catalog downloads from, saved only once SlipStream reads a catalog there. */
export function CatalogAddress({ catalog }: { catalog: IndexersIndexerCatalog }) {
  const save = useSave();
  const form = useAppForm({
    defaultValues: { address: catalog.address },
    validators: { onSubmit: addressSchema },
    onSubmit: async ({ value, formApi }) => {
      try {
        await save(addressSchema.parse(value).address);
      } catch (error) {
        formApi.setErrorMap({ onSubmit: refusal(error) });
        return;
      }
      toast("Saved the Indexer catalog's address");
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
              data
              description="SlipStream downloads Prowlarr's catalog unless you change this. Saving checks that the address serves a catalog first."
            />
          )}
        </form.AppField>
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          {!catalog.addressIsDefault && <UseDefault />}
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}
