import type { IndexersIndexer, NodesNode } from "../api/gen/model";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Select } from "../ui/select";
import type { SignInStep } from "./add/use-add";
import { CaptchaField } from "./fields/captcha";
import { searchingNodes } from "./move-edit";
import { type MoveForm, useMoveForm } from "./move-form";
import { nodeName } from "./words";

/** A cookie pasted for the old home works only there, so a move takes one for the new home. */
function CookieFields({ form, name, to }: { form: MoveForm; name: string; to: string }) {
  return (
    <>
      <p>
        The browser cookie pasted for {name} works only from the Node it was pasted for. Paste the
        cookie of a browser signed in to {name} on {to}&apos;s network.
      </p>
      <form.AppField
        name="cookie"
        validators={{
          onSubmit: ({ value }) =>
            value.trim() === "" ? `Paste the cookie of a browser signed in to ${name}.` : undefined,
        }}
      >
        {(field) => (
          <field.TextArea
            label="Cookie"
            rows={3}
            data
            spellCheck={false}
            description={`The Cookie header of a request to ${name}, from the developer tools of that browser.`}
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

/** What a move to Node id to needs besides the Node: a warning, a new cookie, the captcha's code. */
function Moving({
  form,
  indexer,
  nodes,
  to,
  step,
}: {
  form: MoveForm;
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
  to: string;
  step: SignInStep;
}) {
  const name = nodeName(nodes, to);
  return (
    <>
      {indexer.signsIn && (
        <p className="text-warn">
          {indexer.name} will see a sign-in from {name}&apos;s address. It may ask for a captcha or
          email you about it.
        </p>
      )}
      {indexer.session?.pasted === true && (
        <CookieFields form={form} name={indexer.name} to={name} />
      )}
      {step.kind === "shown" && step.signIn === to && (
        <CaptchaField
          form={form}
          fields={{ captcha: "captcha" }}
          name={indexer.name}
          home={name}
          step={step}
        />
      )}
    </>
  );
}

/** Cancel, and Move, which waits for a Node other than the Indexer's home. */
function MoveButtons({
  form,
  home,
  onCancel,
}: {
  form: MoveForm;
  home: string;
  onCancel: () => void;
}) {
  return (
    <div className="flex flex-wrap justify-end gap-2">
      <Button onClick={onCancel}>Cancel</Button>
      <form.Subscribe selector={(state) => state.values.to === home}>
        {(staying) =>
          staying ? (
            <Button look="primary" disabled>
              Move
            </Button>
          ) : (
            <form.SubmitButton label="Move" />
          )
        }
      </form.Subscribe>
    </div>
  );
}

function MoveFields({
  indexer,
  nodes,
  onDone,
}: {
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
  onDone: () => void;
}) {
  const { form, step } = useMoveForm(indexer, nodes, onDone);
  const options = searchingNodes(nodes).map((node) => ({ value: node.id, label: node.name }));
  return (
    <form
      noValidate
      className="flex flex-col gap-4 text-ink-2"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <p>
          An Indexer searches, signs in and fetches from its home Node. Moving it tests it from the
          new one first.
        </p>
        <form.Field name="to">
          {(field) => (
            <Select
              label="Search from"
              options={options}
              value={field.state.value}
              onValueChange={field.handleChange}
            />
          )}
        </form.Field>
        <form.Subscribe selector={(state) => state.values.to}>
          {(to) =>
            to === indexer.homeNodeId ? null : (
              <Moving form={form} indexer={indexer} nodes={nodes} to={to} step={step} />
            )
          }
        </form.Subscribe>
        <form.FormAlert />
        <MoveButtons form={form} home={indexer.homeNodeId} onCancel={onDone} />
      </form.AppForm>
    </form>
  );
}

export function MoveIndexerDialog({
  indexer,
  nodes,
  open,
  onOpenChange,
}: {
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title={`Move ${indexer.name}`} open={open} onOpenChange={onOpenChange}>
      <MoveFields
        indexer={indexer}
        nodes={nodes}
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}
