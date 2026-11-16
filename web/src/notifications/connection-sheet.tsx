import { useState } from "react";

import type {
  NotificationsConnection as Connection,
  NotificationsEvent as Event,
  NotificationsProvider as Provider,
} from "../api/gen/model";
import { TestLine } from "../settings/test-line";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { ConnectionFields } from "./connection-fields";
import type { NodeChoice } from "./provider-field";
import { ProviderStep } from "./provider-step";
import { type Opened, useConnectionSheet } from "./use-connection-sheet";
import { labelOf } from "./words";

type Choices = { events: readonly Event[]; nodes: readonly NodeChoice[] };

function ConnectionForm({
  opened,
  events,
  nodes,
  onDone,
}: Choices & { opened: Opened; onDone: () => void }) {
  const sheet = useConnectionSheet(opened, events, onDone);
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
        <ConnectionFields
          sheet={sheet}
          events={events}
          nodes={nodes}
          saved={opened.kind === "edit" ? opened.connection : undefined}
        />
        {sheet.outcome !== undefined && <TestLine outcome={sheet.outcome} />}
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

/** Adds a connection: the provider first, then its fields. A Dialog, a bottom sheet on phone. */
export function AddConnectionButton({
  providers,
  ...choices
}: Choices & { providers: readonly Provider[] }) {
  const [open, setOpen] = useState(false);
  const [provider, setProvider] = useState<Provider>();
  const openChange = (next: boolean) => {
    setOpen(next);
    setProvider(undefined);
  };
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          openChange(true);
        }}
      >
        Add a connection
      </Button>
      <Dialog
        title={provider === undefined ? "Add a connection" : `Add ${provider.label}`}
        size={680}
        open={open}
        onOpenChange={openChange}
      >
        {provider === undefined ? (
          <ProviderStep providers={providers} onNext={setProvider} />
        ) : (
          <ConnectionForm
            {...choices}
            opened={{ kind: "add", provider }}
            onDone={() => {
              openChange(false);
            }}
          />
        )}
      </Dialog>
    </>
  );
}

/** Edits a saved connection in the same sheet. */
export function EditConnectionButton({
  connection,
  provider,
  ...choices
}: Choices & { connection: Connection; provider: Provider }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="small"
        aria-label={`Edit ${labelOf(provider, connection)}`}
        onClick={() => {
          setOpen(true);
        }}
      >
        Edit
      </Button>
      <Dialog
        title={`Edit ${labelOf(provider, connection)}`}
        size={680}
        open={open}
        onOpenChange={setOpen}
      >
        <ConnectionForm
          {...choices}
          opened={{ kind: "edit", connection, provider }}
          onDone={() => {
            setOpen(false);
          }}
        />
      </Dialog>
    </>
  );
}
