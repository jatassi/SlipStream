import { Link } from "@tanstack/react-router";

import type { IndexersDefinitionField } from "../../api/gen/model";
import { Checkbox } from "../../ui/checkbox";
import { Select } from "../../ui/select";
import type { AddForm } from "../add/use-add";

/**
 * An info field's text as the definition's authors wrote it, which may be
 * HTML. Parsing it into a document that never renders runs none of it.
 */
export function helpText(help: string): string {
  return new DOMParser().parseFromString(help, "text/html").body.textContent.trim();
}

function Help({ label, help }: { label: string; help: string | null }) {
  if (help === null || helpText(help) === "") {
    return null;
  }
  return (
    <div className="flex flex-col gap-1">
      {label !== "" && <p className="font-semibold text-ink-2">{label}</p>}
      <p className="max-w-[64ch] text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        {helpText(help)}
      </p>
    </div>
  );
}

/**
 * The sentence that stands in for a definition's FlareSolverr hint: the home
 * Node needs a Challenge solver, with a link to that Node.
 */
export function SolverHint({
  name,
  home,
}: {
  name: string;
  home: { readonly id: string; readonly name: string } | undefined;
}) {
  if (home === undefined) {
    return null;
  }
  return (
    <p className="max-w-[64ch] text-ink-2">
      {name} is behind a browser challenge. {home.name} needs a Challenge solver to get past it, or
      a cookie pasted from a browser on the same network.{" "}
      <Link
        to="/nodes/$nodeId/searching"
        params={{ nodeId: home.id }}
        className="rounded-item font-medium text-ink-1 underline-offset-4 hover:underline"
      >
        Set a Challenge solver on {home.name}
      </Link>
    </p>
  );
}

type FieldProps = { form: AddForm; field: IndexersDefinitionField; index: number };

function TextValue({ form, field, index }: FieldProps) {
  return (
    <form.AppField name={`values[${index}]`}>
      {(bound) =>
        field.sealed || field.type === "password" ? (
          <bound.SecretField label={field.label} saved={false} />
        ) : (
          <bound.TextField label={field.label} autoComplete="off" spellCheck={false} />
        )
      }
    </form.AppField>
  );
}

function CheckboxValue({ form, field, index }: FieldProps) {
  return (
    <form.Field name={`values[${index}]`}>
      {(bound) => (
        <Checkbox
          label={field.label}
          checked={bound.state.value === "true"}
          onCheckedChange={(checked) => {
            bound.handleChange(String(checked));
          }}
        />
      )}
    </form.Field>
  );
}

function SelectValue({ form, field, index }: FieldProps) {
  return (
    <form.Field name={`values[${index}]`}>
      {(bound) => (
        <Select
          label={field.label}
          options={field.options}
          value={bound.state.value}
          onValueChange={bound.handleChange}
        />
      )}
    </form.Field>
  );
}

/**
 * One of a definition's settings, drawn by its Cardigann type. A setting
 * whose value is sealed is typed like a password, since it is never shown
 * again. The captcha has its own step, and the FlareSolverr hint is the
 * form's SolverHint, so neither is drawn here.
 */
export function DefinitionField(props: FieldProps) {
  const { field } = props;
  switch (field.type) {
    case "text":
    case "password":
      return <TextValue {...props} />;
    case "checkbox":
      return <CheckboxValue {...props} />;
    case "select":
      return <SelectValue {...props} />;
    case "info":
    case "info_cookie":
    case "info_useragent":
    case "info_category_8000":
      return <Help label={field.label} help={field.help} />;
    case "info_flaresolverr":
    case "cardigannCaptcha":
      return null;
    default:
      return field.type satisfies never;
  }
}
