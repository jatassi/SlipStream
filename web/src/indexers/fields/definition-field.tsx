import { Link } from "@tanstack/react-router";

import type { IndexersDefinitionField, IndexersDefinitionFieldType } from "../../api/gen/model";
import { Checkbox } from "../../ui/checkbox";
import { withFieldGroup } from "../../ui/form";
import { Select } from "../../ui/select";
import { ChoiceSet } from "../add/families-field";
import { chosenOf, numberError, rangeWords } from "./settings";

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

const description = (field: IndexersDefinitionField) => {
  const range = field.type === "number" ? rangeWords(field) : undefined;
  return (
    [
      field.help === null ? "" : helpText(field.help),
      range === undefined ? "" : `${range.charAt(0).toUpperCase()}${range.slice(1)}.`,
    ]
      .filter((part) => part !== "")
      .join(" ") || undefined
  );
};

type One = { field: IndexersDefinitionField; saved: boolean };

const one: One = {
  field: {
    name: "",
    label: "",
    type: "text",
    default: null,
    help: null,
    min: null,
    max: null,
    options: [],
    sealed: false,
  },
  saved: false,
};

const TextSetting = withFieldGroup({
  defaultValues: { value: "" },
  props: one,
  render: function Text({ group, field, saved }) {
    return (
      <group.AppField name="value">
        {(bound) =>
          field.sealed || field.type === "password" ? (
            <bound.SecretField
              label={field.label}
              saved={saved && field.sealed}
              description={description(field)}
            />
          ) : (
            <bound.TextField
              label={field.label}
              autoComplete="off"
              spellCheck={false}
              description={description(field)}
            />
          )
        }
      </group.AppField>
    );
  },
});

const NumberSetting = withFieldGroup({
  defaultValues: { value: "" },
  props: one,
  render: function NumberValue({ group, field }) {
    return (
      <group.AppField
        name="value"
        validators={{ onSubmit: ({ value }) => numberError(field, value) }}
      >
        {(bound) => (
          <bound.TextField
            label={field.label}
            inputMode="numeric"
            autoComplete="off"
            description={description(field)}
          />
        )}
      </group.AppField>
    );
  },
});

/** A multiselect's choices as checkboxes, saved in the options' order. */
const MultiSetting = withFieldGroup({
  defaultValues: { value: "" },
  props: one,
  render: function Multi({ group, field }) {
    return (
      <group.Field name="value">
        {(bound) => (
          <ChoiceSet
            legend={field.label}
            options={field.options}
            chosen={chosenOf(bound.state.value)}
            errors={bound.state.meta.errors}
            description={description(field)}
            onChange={(chosen) => {
              bound.handleChange(
                field.options
                  .filter((option) => chosen.includes(option.value))
                  .map((option) => option.value)
                  .join(","),
              );
            }}
          />
        )}
      </group.Field>
    );
  },
});

const CheckSetting = withFieldGroup({
  defaultValues: { value: "" },
  props: one,
  render: function Check({ group, field }) {
    return (
      <group.Field name="value">
        {(bound) => (
          <Checkbox
            label={field.label}
            checked={bound.state.value === "true"}
            onCheckedChange={(checked) => {
              bound.handleChange(String(checked));
            }}
          />
        )}
      </group.Field>
    );
  },
});

const SelectSetting = withFieldGroup({
  defaultValues: { value: "" },
  props: one,
  render: function Choose({ group, field }) {
    return (
      <group.Field name="value">
        {(bound) => (
          <Select
            label={field.label}
            options={field.options}
            value={bound.state.value}
            onValueChange={bound.handleChange}
          />
        )}
      </group.Field>
    );
  },
});

/** How each field type is drawn: a control, its info text, or nothing here. */
const drawnBy = {
  text: TextSetting,
  password: TextSetting,
  number: NumberSetting,
  multiselect: MultiSetting,
  checkbox: CheckSetting,
  select: SelectSetting,
  info: "help",
  info_cookie: "help",
  info_useragent: "help",
  info_category_8000: "help",
  info_flaresolverr: null,
  cardigannCaptcha: null,
} as const satisfies Record<IndexersDefinitionFieldType, unknown>;

/**
 * A definition's settings in its order, each drawn by its type. A sealed
 * setting is typed like a password, since it is never shown again, and once
 * saved it reads Saved with Replace. The captcha has its own step, and the
 * FlareSolverr hint is the form's SolverHint, so neither is drawn here.
 */
export const SettingFields = withFieldGroup({
  defaultValues: { values: [] as string[] },
  props: { settings: [] as readonly IndexersDefinitionField[], saved: false },
  render: function Settings({ group, settings, saved }) {
    return settings.map((field, index) => {
      const Control = drawnBy[field.type];
      if (Control === null) {
        return null;
      }
      if (Control === "help") {
        return <Help key={field.name} label={field.label} help={field.help} />;
      }
      return (
        <Control
          key={field.name}
          form={group}
          fields={{ value: `values[${index}]` }}
          field={field}
          saved={saved}
        />
      );
    });
  },
});
