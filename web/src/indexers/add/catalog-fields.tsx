import type { IndexersDefinitionForm, NodesNode } from "../../api/gen/model";
import { AddressField } from "../fields/address-field";
import { SettingFields, SolverHint } from "../fields/definition-field";
import type { PortForm } from "../port-form";
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

/**
 * The User-Agent field's help: a catalog definition can't say whether its
 * site needs one, and a ported site that doesn't take one gets no field.
 */
function userAgentHelp({ entry, port }: IndexersDefinitionForm): string | undefined {
  if (port === null) {
    return "The User-Agent of the same browser. Some Trackers accept the cookie only with it.";
  }
  return port.userAgent
    ? `Use the User-Agent of the browser you copied the cookie from. ${entry.name} accepts the cookie only with it.`
    : undefined;
}

/** What adding a ported site adds to its catalog-shaped lead. */
export function portLead(form: PortForm, home: string): string[] {
  const { name } = form.entry;
  return [
    catalogLead(form, home),
    `SlipStream includes support for ${name}. Fixes to it come with SlipStream updates.`,
    ...(form.port.recent ? [] : [`${name} has no list of new Releases, so RSS sync can't use it.`]),
  ];
}

function CookieFields({
  form,
  name,
  userAgent,
}: {
  form: AddForm;
  name: string;
  userAgent: string | undefined;
}) {
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
      {userAgent !== undefined && (
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
      )}
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
        saved={[]}
      />
      {definition.cookie && (
        <CookieFields form={form} name={name} userAgent={userAgentHelp(definition)} />
      )}
      {definition.solver && <SolverHint name={name} home={home} />}
    </>
  );
}
