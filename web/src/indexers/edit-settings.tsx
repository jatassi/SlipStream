import type { IndexersIndexer } from "../api/gen/model";
import { userAgentHelp } from "./add/catalog-fields";
import type { SettingsForm } from "./definition-form";
import type { EditForm } from "./edit-form";
import { pinOf } from "./edit-values";
import { AddressField } from "./fields/address-field";
import { SettingFields } from "./fields/definition-field";

/**
 * A new browser cookie for a site that signs in with one, Saved while the
 * Indexer holds a pasted one, and the User-Agent beside it once one is
 * pasted, when the site takes one.
 */
function CookieFields({
  form,
  name,
  pasted,
  userAgent,
}: {
  form: EditForm;
  name: string;
  pasted: boolean;
  userAgent: string | undefined;
}) {
  return (
    <>
      <form.AppField name="cookie">
        {(field) => (
          <field.SecretField
            label="Cookie"
            saved={pasted}
            description={`The Cookie header of a request to ${name}, from the developer tools of a browser signed in to it.`}
          />
        )}
      </form.AppField>
      {userAgent !== undefined && (
        <form.Subscribe selector={(state) => state.values.cookie !== ""}>
          {(pasting) =>
            pasting && (
              <form.AppField name="userAgent">
                {(field) => (
                  <field.TextField
                    label="User-Agent"
                    data
                    autoComplete="off"
                    spellCheck={false}
                    description={userAgent}
                  />
                )}
              </form.AppField>
            )
          }
        </form.Subscribe>
      )}
    </>
  );
}

/**
 * A catalog or native Indexer's address and settings, drawn as Add Indexer
 * draws them, with each sealed one that holds a value Saved until replaced,
 * then the browser cookie of a site that signs in with one.
 */
export function EditSettings({
  form,
  settings,
  indexer,
}: {
  form: EditForm;
  settings: SettingsForm;
  indexer: IndexersIndexer;
}) {
  const { links, fields, entry, cookie } = settings.form;
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
      <SettingFields
        form={form}
        fields={{ values: "values" }}
        settings={fields}
        saved={pinOf(indexer)?.sealed ?? []}
      />
      {cookie && (
        <CookieFields
          form={form}
          name={entry.name}
          pasted={indexer.session?.pasted === true}
          userAgent={userAgentHelp(settings.form)}
        />
      )}
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        {cookie
          ? "Saving a new address, setting or cookie tests the Indexer first."
          : "Saving a new address or setting tests the Indexer first."}
      </p>
    </>
  );
}
