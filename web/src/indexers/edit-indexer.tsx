import type { IndexersIndexer } from "../api/gen/model";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Skeleton } from "../ui/skeleton";
import { type SettingsForm, useSettingsForm } from "./definition-form";
import { type EditForm, useEditForm } from "./edit-form";
import { EditSettings } from "./edit-settings";
import { pinOf } from "./edit-values";
import { CodeField } from "./fields/captcha";
import { LimitField } from "./fields/limit-field";

function Secrets({ form, indexer }: { form: EditForm; indexer: IndexersIndexer }) {
  if (indexer.kind === "catalog" || indexer.kind === "native") {
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

function EditFields({
  indexer,
  settings,
  onDone,
}: {
  indexer: IndexersIndexer;
  settings: SettingsForm | undefined;
  onDone: () => void;
}) {
  const { form, codeShown } = useEditForm(indexer, settings, onDone);
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
        {settings !== undefined && (
          <EditSettings form={form} settings={settings} sealed={pinOf(indexer)?.sealed ?? []} />
        )}
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
        {codeShown && <CodeField form={form} fields={{ code: "code" }} name={indexer.name} />}
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          <Button onClick={onDone}>Cancel</Button>
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}

/** The form once a catalog or native Indexer's settings form is read; a feed has none to read. */
function Loaded({ indexer, onDone }: { indexer: IndexersIndexer; onDone: () => void }) {
  const settings = useSettingsForm(indexer);
  if (settings.isPending && settings.fetchStatus !== "idle") {
    return <Skeleton label="Loading the settings" />;
  }
  return <EditFields indexer={indexer} settings={settings.data} onDone={onDone} />;
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
      <Loaded
        indexer={indexer}
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}
