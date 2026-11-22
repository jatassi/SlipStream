import { useState } from "react";

import type { DownloadclientsKind as Kind } from "../api/gen/model";
import { TestLine, type TestOutcome } from "../settings/test-line";
import { Button, type ButtonLook } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Switch } from "../ui/switch";
import { ClientDetails } from "./client-details";
import { LeadFields, type NodeChoice } from "./client-fields";
import { type TestResult, daemonsOf, kindOf } from "./client-form";
import { type ClientSheetState, type Opened, useClientSheet } from "./use-client-sheet";

type Props = {
  opened: Opened;
  kinds: readonly Kind[];
  /** The downloading Nodes a client may run on. */
  nodes: readonly NodeChoice[];
  onDone: () => void;
};

const outcomeOf = (result: TestResult, node: string): TestOutcome =>
  result.passed
    ? {
        passed: true,
        sentence: `The Test passed. ${node} connected to ${result.report.product} ${result.report.version}.`,
        warnings: result.report.warnings,
      }
    : { passed: false, sentence: result.sentence };

const nameOf = (nodes: readonly NodeChoice[], id: string): string =>
  nodes.find((node) => node.id === id)?.name ?? "its Node";

function Fields({
  sheet,
  kinds,
  nodes,
  changingNode,
}: { sheet: ClientSheetState; changingNode: boolean } & Omit<Props, "opened" | "onDone">) {
  const { form, saved, result } = sheet;
  const picksNode =
    nodes.length > 1 && (saved === undefined || saved.torrents === 0 || changingNode);
  return (
    <form.Subscribe selector={(state) => [state.values.kind, state.values.nodeId] as const}>
      {([kindId, nodeId]) => {
        const kind = kindOf(kinds, kindId);
        const node = nameOf(nodes, nodeId);
        return (
          <>
            <LeadFields
              form={form}
              kind={kind}
              kinds={kinds}
              nodes={nodes}
              picksNode={picksNode}
              picksKind={saved === undefined}
              saved={{
                password: saved?.passwordSaved ?? false,
                apiKey: saved?.apiKeySaved ?? false,
              }}
            />
            <ClientDetails
              form={form}
              kind={kind}
              node={node}
              saved={saved}
              daemons={daemonsOf(result)}
              open={sheet.detailsOpen}
              onOpenChange={sheet.setDetailsOpen}
            />
            {result !== undefined && <TestLine outcome={outcomeOf(result, node)} />}
          </>
        );
      }}
    </form.Subscribe>
  );
}

function ClientForm({ opened, kinds, nodes, onDone }: Props) {
  const sheet = useClientSheet(opened, kinds, onDone);
  const changingNode = opened.kind === "edit" && opened.changingNode === true;
  const { form } = sheet;
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <form.Field name="on">
          {(field) => (
            <div className="flex justify-end">
              <Switch label="On" checked={field.state.value} onCheckedChange={field.handleChange} />
            </div>
          )}
        </form.Field>
        {changingNode && (
          <p className="text-ink-2">
            Pick the Node it runs on now and each Client folder&apos;s path there. That Node first
            checks that it finds every torrent&apos;s files, then the client and its torrents move
            to it. Nothing is copied.
          </p>
        )}
        <Fields sheet={sheet} kinds={kinds} nodes={nodes} changingNode={changingNode} />
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          <Button onClick={onDone}>Cancel</Button>
          <Button busy={sheet.testing} onClick={() => void sheet.test()}>
            Test
          </Button>
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}

/** The add and edit sheet of an external Download Client: a Dialog, a bottom sheet on phone. */
export function ClientSheet({
  open,
  onOpenChange,
  ...props
}: Omit<Props, "onDone"> & { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { opened } = props;
  const title =
    opened.kind === "add"
      ? "Add a Download Client"
      : `${opened.changingNode === true ? "Change the Node of" : "Edit"} ${opened.client.name}`;
  return (
    <Dialog title={title} size={680} open={open} onOpenChange={onOpenChange}>
      <ClientForm
        {...props}
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}

/** A button that opens the client sheet, labelled for what it opens on. */
export function ClientSheetButton({
  label,
  look = "default",
  ...props
}: Omit<Props, "onDone"> & { label: string; look?: ButtonLook }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="small"
        look={look}
        onClick={() => {
          setOpen(true);
        }}
      >
        {label}
      </Button>
      <ClientSheet {...props} open={open} onOpenChange={setOpen} />
    </>
  );
}
