import type { ReactNode } from "react";

import type {
  NotificationsField as ProviderFieldSpec,
  NotificationsFieldKind as FieldKind,
} from "../api/gen/model";
import { errorText } from "../ui/form-context";
import { Select } from "../ui/select";
import { Switch } from "../ui/switch";
import type { ConnectionForm } from "./use-connection-sheet";

export type NodeChoice = { readonly id: string; readonly name: string };

/** What a field is drawn with: the form, the provider's field, and the Nodes a script may run on. */
type Drawn = {
  form: ConnectionForm;
  field: ProviderFieldSpec;
  nodes: readonly NodeChoice[];
};

const help = (field: ProviderFieldSpec) => (field.help === "" ? undefined : field.help);

const withHint = (field: ProviderFieldSpec, hint: string) =>
  field.help === "" ? hint : `${field.help} ${hint}`;

function TextKind({ form, field, kind }: Drawn & { kind: "text" | "url" | "number" }) {
  return (
    <form.AppField name={`settings.${field.key}`}>
      {(control) => (
        <control.TextField
          label={field.label}
          type={kind === "number" ? "number" : kind}
          data={kind === "url"}
          autoComplete="off"
          spellCheck={false}
          description={help(field)}
        />
      )}
    </form.AppField>
  );
}

function SelectKind({ form, field }: Drawn) {
  return (
    <form.Field name={`settings.${field.key}`}>
      {(control) => (
        <Select
          label={field.label}
          options={field.options ?? []}
          value={String(control.state.value)}
          onValueChange={control.handleChange}
        />
      )}
    </form.Field>
  );
}

function SwitchKind({ form, field }: Drawn) {
  return (
    <form.Field name={`settings.${field.key}`}>
      {(control) => (
        <div className="flex flex-col gap-1">
          <Switch
            label={field.label}
            checked={control.state.value === true}
            onCheckedChange={control.handleChange}
          />
          {field.help !== "" && <p className="text-meta text-ink-3">{field.help}</p>}
        </div>
      )}
    </form.Field>
  );
}

function ListKind({ form, field }: Drawn) {
  return (
    <form.AppField name={`settings.${field.key}`}>
      {(control) => (
        <control.TextArea
          label={field.label}
          rows={3}
          data
          spellCheck={false}
          description={withHint(field, "Put each one on its own line.")}
        />
      )}
    </form.AppField>
  );
}

function NodeKind({ form, field, nodes }: Drawn) {
  return (
    <form.Field name="node">
      {(control) => (
        <div className="flex flex-col gap-1.5">
          <Select
            label="Runs on"
            options={nodes.map((node) => ({ value: node.id, label: node.name }))}
            value={control.state.value}
            onValueChange={control.handleChange}
          />
          {errorText(control.state.meta.errors) === undefined ? (
            field.help !== "" && <p className="text-meta text-ink-3">{field.help}</p>
          ) : (
            <p role="alert" className="text-meta text-fail-ink">
              {errorText(control.state.meta.errors)}
            </p>
          )}
        </div>
      )}
    </form.Field>
  );
}

function DurationKind({ form, field }: Drawn) {
  return (
    <form.AppField name={`settings.${field.key}`}>
      {(control) => (
        <control.TextField
          label={field.label}
          type="number"
          inputMode="numeric"
          autoComplete="off"
          description={withHint(field, `Use ${field.min ?? 0} to ${field.max ?? 0} seconds.`)}
        />
      )}
    </form.AppField>
  );
}

const drawnBy = {
  text: (props) => <TextKind {...props} kind="text" />,
  url: (props) => <TextKind {...props} kind="url" />,
  number: (props) => <TextKind {...props} kind="number" />,
  select: SelectKind,
  switch: SwitchKind,
  list: ListKind,
  headers: ListKind,
  node: NodeKind,
  duration: DurationKind,
} satisfies Record<FieldKind, (props: Drawn) => ReactNode>;

const secretRows: Partial<Record<FieldKind, number>> = { list: 3, headers: 3 };

const secretHint: Partial<Record<FieldKind, string>> = {
  list: "Put each one on its own line.",
  headers: "Put each header on its own line, as its name, a colon and its value.",
};

function SecretKind({
  form,
  field,
  sealed,
  shown,
}: Drawn & { sealed: boolean; shown: string | undefined }) {
  const hint = secretHint[field.kind];
  return (
    <form.AppField name={`secrets.${field.key}`}>
      {(control) => (
        <control.SecretField
          label={field.label}
          saved={sealed}
          shown={shown}
          type={field.kind === "url" ? "url" : "password"}
          removable={field.optional}
          rows={secretRows[field.kind]}
          description={hint === undefined ? help(field) : withHint(field, hint)}
        />
      )}
    </form.AppField>
  );
}

/** One provider field, drawn by its kind. A secret reads Saved once saved, with Replace, and Remove when it is optional. */
export function ProviderField({
  sealed,
  shown,
  ...props
}: Drawn & { sealed: boolean; shown: string | undefined }) {
  if (props.field.secret) {
    return <SecretKind {...props} sealed={sealed} shown={shown} />;
  }
  const Draw = drawnBy[props.field.kind];
  return <Draw {...props} />;
}
