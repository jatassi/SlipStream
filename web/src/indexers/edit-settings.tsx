import type { SettingsForm } from "./definition-form";
import type { EditForm } from "./edit-form";
import { AddressField } from "./fields/address-field";
import { SettingFields } from "./fields/definition-field";

/**
 * A catalog or native Indexer's address and settings, drawn as Add Indexer
 * draws them, with each sealed one Saved until replaced.
 */
export function EditSettings({ form, settings }: { form: EditForm; settings: SettingsForm }) {
  const { links, fields, entry } = settings.form;
  const ported = settings.kind === "native";
  return (
    <>
      {(ported || links.length > 1) && (
        <AddressField
          form={form}
          fields="baseAddress"
          name={entry.name}
          links={links}
          typeable={ported}
        />
      )}
      <SettingFields form={form} fields={{ values: "values" }} settings={fields} saved />
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        Saving a new address or setting tests the Indexer first.
      </p>
    </>
  );
}
