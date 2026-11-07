import type { IndexersDefinitionForm, NodesNode } from "../../api/gen/model";
import { DefinitionField, SolverHint } from "../fields/definition-field";
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

function CookieFields({ form, name }: { form: AddForm; name: string }) {
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
            description="The User-Agent of the same browser. Some Trackers accept the cookie only with it."
          />
        )}
      </form.AppField>
    </>
  );
}

/** The definition's settings in its order, then a pasted cookie and the Challenge solver hint it needs. */
export function CatalogFields({
  form,
  definition,
  home,
}: {
  form: AddForm;
  definition: IndexersDefinitionForm;
  home: NodesNode | undefined;
}) {
  return (
    <>
      {definition.fields.map((field, index) => (
        <DefinitionField key={field.name} form={form} field={field} index={index} />
      ))}
      {definition.cookie && <CookieFields form={form} name={definition.entry.name} />}
      {definition.solver && <SolverHint name={definition.entry.name} home={home} />}
    </>
  );
}
