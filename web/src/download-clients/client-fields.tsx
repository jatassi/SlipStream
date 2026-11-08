import type { DownloadclientsKind as Kind } from "../api/gen/model";
import { Select } from "../ui/select";
import { examples, kindOf, signInOf } from "./client-form";
import type { ClientForm } from "./use-client-sheet";

export type NodeChoice = { readonly id: string; readonly name: string };

function addressHint(kind: Kind): string {
  const example = `The whole address, such as ${examples[kind.kind]}.`;
  return kind.path === "" ? example : `${example} With no path, SlipStream adds ${kind.path}.`;
}

function NodeField({ form, nodes }: { form: ClientForm; nodes: readonly NodeChoice[] }) {
  return (
    <form.Field name="nodeId">
      {(field) => (
        <Select
          label="Node"
          options={nodes.map((node) => ({ value: node.id, label: node.name }))}
          value={field.state.value}
          onValueChange={field.handleChange}
        />
      )}
    </form.Field>
  );
}

function KindField({ form, kinds }: { form: ClientForm; kinds: readonly Kind[] }) {
  return (
    <form.Field name="kind">
      {(field) => (
        <Select
          label="Client"
          options={kinds.map((kind) => ({ value: kind.kind, label: kind.name }))}
          value={field.state.value}
          onValueChange={(next) => {
            const was = kindOf(kinds, field.state.value);
            if (form.state.values.name.trim() === was.name) {
              form.setFieldValue("name", kindOf(kinds, next).name);
            }
            field.handleChange(next);
          }}
        />
      )}
    </form.Field>
  );
}

type Saved = { readonly password: boolean; readonly apiKey: boolean };

/** The sign-in fields kind takes. A saved secret reads Saved with Replace. */
function SignInFields({ form, kind, saved }: { form: ClientForm; kind: Kind; saved: Saved }) {
  const signIn = signInOf(kind);
  return (
    <>
      {signIn.username && (
        <form.AppField name="username">
          {(field) => (
            <field.TextField
              label="Username"
              autoComplete="off"
              spellCheck={false}
              {...(kind.password === "optional" && {
                description: `Leave it empty when ${kind.name} has no sign-in set up.`,
              })}
            />
          )}
        </form.AppField>
      )}
      {signIn.password && (
        <form.AppField name="password">
          {(field) => <field.SecretField label="Password" saved={saved.password} />}
        </form.AppField>
      )}
      {signIn.apiKey && (
        <form.AppField name="apiKey">
          {(field) => (
            <field.SecretField
              label="API key"
              saved={saved.apiKey}
              description={`${kind.name} 5.2 and later take an API key. SlipStream then signs in with it in place of the username and password.`}
            />
          )}
        </form.AppField>
      )}
    </>
  );
}

/** What the sheet leads with: where the client runs, what it is, its address and how SlipStream signs in. */
export function LeadFields({
  form,
  kind,
  kinds,
  nodes,
  picksNode,
  picksKind,
  saved,
}: {
  form: ClientForm;
  kind: Kind;
  kinds: readonly Kind[];
  nodes: readonly NodeChoice[];
  picksNode: boolean;
  picksKind: boolean;
  saved: Saved;
}) {
  return (
    <>
      {picksNode && <NodeField form={form} nodes={nodes} />}
      {picksKind && <KindField form={form} kinds={kinds} />}
      <form.AppField name="name">
        {(field) => (
          <field.TextField
            label="Name"
            autoComplete="off"
            description="Shown on Torrents and in Health. Rename it to tell two clients apart."
          />
        )}
      </form.AppField>
      <form.AppField name="address">
        {(field) => (
          <field.TextField
            label="Address"
            type="url"
            data
            autoComplete="off"
            spellCheck={false}
            placeholder={examples[kind.kind]}
            description={addressHint(kind)}
          />
        )}
      </form.AppField>
      <SignInFields form={form} kind={kind} saved={saved} />
    </>
  );
}
