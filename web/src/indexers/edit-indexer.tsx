import { useQueryClient } from "@tanstack/react-query";

import type { IndexersIndexer } from "../api/gen/model";
import { updateIndexer } from "../api/gen/query/indexers/indexers";
import { adoptIndexer } from "../live/adopt";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { useAppForm } from "../ui/form";
import { toast } from "../ui/toast";
import { editOf, editSchema, refusal, valuesOf } from "./edit-values";
import { LimitField } from "./fields/limit-field";

function useEditForm(indexer: IndexersIndexer, onDone: () => void) {
  const queryClient = useQueryClient();
  return useAppForm({
    defaultValues: valuesOf(indexer),
    validators: { onSubmit: editSchema },
    onSubmit: async ({ value, formApi }) => {
      const edit = editOf(indexer, value);
      if (Object.keys(edit).length > 0) {
        try {
          adoptIndexer(queryClient, await updateIndexer(indexer.id, edit));
        } catch (error) {
          formApi.setErrorMap({ onSubmit: refusal(error) });
          return;
        }
        toast(`Saved ${edit.name ?? indexer.name}`);
      }
      onDone();
    },
  });
}

type EditForm = ReturnType<typeof useEditForm>;

function Secrets({ form, indexer }: { form: EditForm; indexer: IndexersIndexer }) {
  if (indexer.kind === "catalog") {
    return null;
  }
  return (
    <>
      <form.AppField name="address">
        {(field) => (
          <field.SecretField
            label="Feed address"
            saved
            shown={indexer.feedHost ?? undefined}
            type="url"
            description="The whole address may hold a passkey, so only its host is shown."
          />
        )}
      </form.AppField>
      {indexer.kind === "torznab" && (
        <form.AppField name="apiKey">
          {(field) => <field.SecretField label="API key" saved />}
        </form.AppField>
      )}
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        Saving a new address or API key tests the Indexer first.
      </p>
    </>
  );
}

function EditFields({ indexer, onDone }: { indexer: IndexersIndexer; onDone: () => void }) {
  const form = useEditForm(indexer, onDone);
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
        <form.AppField name="name">
          {(field) => <field.TextField label="Name" autoComplete="off" />}
        </form.AppField>
        <Secrets form={form} indexer={indexer} />
        <form.AppField name="priority">
          {(field) => (
            <field.TextField
              label="Priority"
              inputMode="numeric"
              description="From 1 to 50. When two Releases are equal, the one from the lower number wins."
            />
          )}
        </form.AppField>
        <LimitField form={form} fields="searches" label="Searches" />
        <LimitField form={form} fields="grabs" label="Grabs" />
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          <Button onClick={onDone}>Cancel</Button>
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}

export function EditIndexerDialog({
  indexer,
  open,
  onOpenChange,
}: {
  indexer: IndexersIndexer;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title={`Edit ${indexer.name}`} open={open} onOpenChange={onOpenChange}>
      <EditFields
        indexer={indexer}
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}
