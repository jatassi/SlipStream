import type { IndexersDefinitionForm, NodesNode } from "../../api/gen/model";
import { AddressField } from "../fields/address-field";
import { SettingFields, SolverHint } from "../fields/definition-field";
import type { PortForm } from "../port-shape";
import type { AddForm } from "./use-add";

const privacyLead = {
  private: "a private",
  "semi-private": "a semi-private",
  public: "a public",
} as const;

/** What adding the definition does, in a sentence or two. */
export function catalogLead(form: IndexersDefinitionForm, home: string): string {
  const what = `${form.entry.name} is ${privacyLead[form.entry.privacy]} Tracker.`;
  if (!form.session) {
    return `${what} SlipStream searches it from ${home}.`;
  }
  const signsIn = `${what} SlipStream signs in to it from ${home} and keeps the session there.`;
  return form.captcha
    ? `${signsIn} Its sign-in shows a captcha, which comes after these fields.`
    : signsIn;
}

const userAgentHelp = {
  catalog: () =>
    "The User-Agent of the same browser. Some Trackers accept the cookie only with it.",
  native: (name: string) =>
    `Use the User-Agent of the browser you copied the cookie from. ${name} accepts the cookie only with it.`,
} as const;

/** What adding a ported site adds to its catalog-shaped lead. */
export function portLead(form: PortForm, home: string): string[] {
  const { name } = form.entry;
  return [
    catalogLead(form, home),
    `SlipStream includes support for ${name}. Fixes to it come with SlipStream updates.`,
    ...(form.recent ? [] : [`${name} has no list of new Releases, so RSS sync can't use it.`]),
  ];
}

function CookieFields({ form, name, ported }: { form: AddForm; name: string; ported: boolean }) {
  return (
    <>
      <form.AppField
        name="cookie"
        validators={{
          onSubmit: ({ value }) =>
            value.trim() === ""
              ? "Paste the cookie of a browser signed in to the Tracker."
              : undefined,
        }}
      >
        {(field) => (
          <field.TextArea
            label="Cookie"
            rows={3}
            data
            spellCheck={false}
            description={`The Cookie header of a request to ${name}, from the developer tools of a browser signed in to it.`}
          />
        )}
      </form.AppField>
      <form.AppField name="userAgent">
        {(field) => (
          <field.TextField
            label="User-Agent"
            data
            autoComplete="off"
            spellCheck={false}
            description={ported ? userAgentHelp.native(name) : userAgentHelp.catalog()}
          />
        )}
      </form.AppField>
    </>
  );
}

/**
 * The definition's settings in its order, then a pasted cookie and the
 * Challenge solver hint it needs. A ported site's address leads, since it
 * may be typed.
 */
export function CatalogFields({
  form,
  definition,
  home,
  ported,
}: {
  form: AddForm;
  definition: IndexersDefinitionForm;
  home: NodesNode | undefined;
  ported: boolean;
}) {
  const { name } = definition.entry;
  return (
    <>
      {ported && (
        <AddressField
          form={form}
          fields="baseAddress"
          name={name}
          links={definition.links}
          typeable
        />
      )}
      <SettingFields
        form={form}
        fields={{ values: "values" }}
        settings={definition.fields}
        saved={false}
      />
      {definition.cookie && <CookieFields form={form} name={name} ported={ported} />}
      {definition.solver && <SolverHint name={name} home={home} />}
    </>
  );
}
