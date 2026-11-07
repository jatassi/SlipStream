import { Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { isApiError } from "../../api/errors";
import { addCustomDefinition } from "../../api/gen/query/indexers/indexers";
import { Button } from "../../ui/button";
import { FileField } from "../../ui/file-field";
import { useAppForm } from "../../ui/form";
import { submitErrors } from "../../ui/form-context";
import { toast } from "../../ui/toast";
import { type Refused, RefusedLine } from "./refused-line";

const sentence = (title: string) => (/[.!?]$/u.test(title) ? title : `${title}.`);

/** Adds the definition, then opens its form like any catalog definition's. */
function useCustomForm(setRefused: (refused: Refused | undefined) => void) {
  const navigate = useNavigate();
  return useAppForm({
    defaultValues: { yaml: "" },
    onSubmit: async ({ value, formApi }) => {
      setRefused(undefined);
      try {
        const entry = await addCustomDefinition({ yaml: value.yaml });
        toast(`Added the definition ${entry.name}`);
        await navigate({
          to: "/settings/indexers/new/catalog/$definitionId",
          params: { definitionId: entry.id },
        });
      } catch (error) {
        if (isApiError(error) && error.code !== "invalid") {
          setRefused({ said: sentence(error.title), reason: error.detail });
          return;
        }
        formApi.setErrorMap({ onSubmit: submitErrors(error) });
      }
    },
  });
}

/**
 * A Cardigann definition the Admin pastes or uploads. It is stored on the
 * Controller, so it works from any browser.
 */
export function CustomDefinition() {
  const [refused, setRefused] = useState<Refused>();
  const form = useCustomForm(setRefused);
  return (
    <form
      noValidate
      className="flex max-w-[760px] flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <p className="text-ink-1">
          A custom definition runs a Tracker the Indexer catalog lacks, or a fixed definition before
          the catalog has it. It has to load as a catalog definition does, and its id can&apos;t be
          one the catalog already has.
        </p>
        <FileField
          label="Definition file"
          accept=".yml,.yaml,application/yaml,text/yaml"
          description="A YAML file in Prowlarr's Cardigann format. Its text goes into the field below."
          onFile={async (file) => {
            form.setFieldValue("yaml", await file.text());
          }}
        />
        <form.AppField
          name="yaml"
          validators={{
            onSubmit: ({ value }) =>
              value.trim() === "" ? "Paste the definition, or choose its file." : undefined,
          }}
        >
          {(field) => <field.TextArea label="Definition" rows={16} data spellCheck={false} />}
        </form.AppField>
        {refused !== undefined && <RefusedLine refused={refused} />}
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          <Button nativeButton={false} render={<Link to="/settings/indexers/new" />}>
            Cancel
          </Button>
          <form.SubmitButton label="Add definition" />
        </div>
      </form.AppForm>
    </form>
  );
}
