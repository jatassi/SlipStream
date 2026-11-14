import type { ReactNode } from "react";

import type { MediaserversKindView as Kind, NodesNode } from "../api/gen/model";
import { Details } from "../ui/details";
import { errorText } from "../ui/form-context";
import { Select } from "../ui/select";
import { Refusal } from "../ui/setting-parts";
import { WarnLine } from "./lines";
import {
  type Reaching,
  defaultReaching,
  isLocalhost,
  noNode,
  reachingChoices,
  suggestedRunsOn,
} from "./server-form";
import type { ServerFormState } from "./use-server-form";

const example = "http://192.168.1.20:8096";

const untouched = (meta: { isTouched: boolean } | undefined) => meta?.isTouched !== true;

export function Refused({ errors }: { errors: readonly unknown[] }) {
  const error = errorText(errors);
  return error === undefined ? null : <Refusal error={error} />;
}

/** Which Node reaches the server and which it runs on. A choice the Admin made stays. */
export function NodeFields({
  sheet: { form },
  nodes,
  reaching,
}: {
  sheet: ServerFormState;
  nodes: readonly NodesNode[];
  reaching: Reaching;
}) {
  return (
    <>
      <form.Field name="runsOnNodeId">
        {(field) => (
          <Select
            label="Runs on"
            options={[
              { value: noNode, label: "No Node" },
              ...nodes.map((node) => ({ value: node.id, label: node.name })),
            ]}
            value={field.state.value}
            onValueChange={(next) => {
              field.handleChange(next);
              if (untouched(form.getFieldMeta("reachingNodeId"))) {
                form.setFieldValue("reachingNodeId", defaultReaching(next, reaching), {
                  dontUpdateMeta: true,
                });
              }
            }}
          />
        )}
      </form.Field>
      <form.Field name="reachingNodeId">
        {(field) => (
          <div className="flex flex-col gap-1.5">
            <Select
              label="Reached from"
              options={reachingChoices(reaching).map((node) => ({
                value: node.id,
                label: node.id === reaching.controller.id ? `${node.name} (Controller)` : node.name,
              }))}
              value={field.state.value}
              onValueChange={field.handleChange}
            />
            <Refused errors={field.state.meta.errors} />
          </div>
        )}
      </form.Field>
    </>
  );
}

function LocalhostWarning({
  form,
  reaching,
}: {
  form: ServerFormState["form"];
  reaching: Reaching;
}) {
  return (
    <form.Subscribe selector={(state) => state.values}>
      {({ address, fallbackAddress, reachingNodeId }) => {
        const worker = reaching.workers.find((node) => node.id === reachingNodeId);
        return worker !== undefined && isLocalhost(address) && fallbackAddress.trim() === "" ? (
          <WarnLine
            text={`The Controller can't use localhost on ${worker.name}. Enter ${worker.name}'s network address, or add a fallback address under Details.`}
          />
        ) : null;
      }}
    </form.Subscribe>
  );
}

/** Typing an address suggests the Node the server runs on, until the Admin picks one. */
function AddressField({ sheet: { form }, kind, nodes, reaching }: Omit<Props, "keySaved">) {
  return (
    <form.AppField
      name="address"
      listeners={{
        onChange: ({ value }) => {
          if (!untouched(form.getFieldMeta("runsOnNodeId"))) {
            return;
          }
          const runsOn = suggestedRunsOn(value, nodes);
          form.setFieldValue("runsOnNodeId", runsOn, { dontUpdateMeta: true });
          if (untouched(form.getFieldMeta("reachingNodeId"))) {
            form.setFieldValue("reachingNodeId", defaultReaching(runsOn, reaching), {
              dontUpdateMeta: true,
            });
          }
        },
      }}
    >
      {(field) => (
        <field.TextField
          label="Address"
          type="url"
          data
          autoComplete="off"
          spellCheck={false}
          placeholder={example}
          description={`The whole address, such as ${example}, as the Node that reaches ${kind.label} calls it.`}
        />
      )}
    </form.AppField>
  );
}

/** The server's name, where it is and its key. */
function LeadFields({ sheet, kind, nodes, reaching, keySaved }: Props) {
  const { form } = sheet;
  return (
    <>
      <form.AppField name="name">
        {(field) => (
          <field.TextField
            label="Name"
            autoComplete="off"
            description="Shown in Health and on each Work's Open in button. Rename it to tell two servers apart."
          />
        )}
      </form.AppField>
      <AddressField sheet={sheet} kind={kind} nodes={nodes} reaching={reaching} />
      {kind.signIn === "api_key_or_admin" && (
        <form.AppField name="apiKey">
          {(field) => (
            <field.SecretField
              label="API key"
              saved={keySaved}
              description={`A key from ${kind.label}'s API keys page. ${kind.label} can revoke it there at any time.`}
            />
          )}
        </form.AppField>
      )}
    </>
  );
}

/** The addresses only some setups need. */
function AddressDetails({ sheet, kind }: Pick<Props, "sheet" | "kind">) {
  const { form } = sheet;
  return (
    <Details
      fields
      open={sheet.detailsOpen}
      onOpenChange={sheet.setDetailsOpen}
      summary="SlipStream calls the address from the Node it's reached from, and the fallback address from the Controller while that Node is offline."
    >
      <div className="mt-2 flex flex-col gap-4">
        <form.AppField name="fallbackAddress">
          {(field) => (
            <field.TextField
              label="Fallback address"
              type="url"
              data
              autoComplete="off"
              spellCheck={false}
              description="What the Controller calls. Leave it empty when the Controller can use the address."
            />
          )}
        </form.AppField>
        {kind.browser && (
          <form.AppField name="browserAddress">
            {(field) => (
              <field.TextField
                label="Browser address"
                type="url"
                data
                autoComplete="off"
                spellCheck={false}
                description={`Where Open in takes a browser to ${kind.label}. Leave it empty for no Open in button.`}
              />
            )}
          </form.AppField>
        )}
      </div>
    </Details>
  );
}

type Props = {
  sheet: ServerFormState;
  kind: Kind;
  nodes: readonly NodesNode[];
  reaching: Reaching;
  keySaved: boolean;
};

/**
 * A server's fields: what it's called, where it is, its key and the Nodes
 * that reach it, with the addresses only some setups need under Details.
 */
export function ServerFields({
  signIn,
  ...props
}: Props & {
  /** Sits under the key field, such as the administrator sign-in. */
  signIn?: ReactNode;
}) {
  return (
    <>
      <LeadFields {...props} />
      {signIn}
      <NodeFields sheet={props.sheet} nodes={props.nodes} reaching={props.reaching} />
      <LocalhostWarning form={props.sheet.form} reaching={props.reaching} />
      <AddressDetails sheet={props.sheet} kind={props.kind} />
    </>
  );
}
