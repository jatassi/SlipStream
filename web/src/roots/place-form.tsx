import { type ReactNode, useState } from "react";

import { DialogActions } from "../account/dialog-actions";
import { FolderField } from "../folders/folder-field";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import { NodeSelect, useStorageNodes } from "./storage-nodes";
import type { Editing } from "./use-place";

function PlaceFields({ editing }: { editing: Editing }) {
  const { storage } = useStorageNodes();
  const { place, setNodeId, check, verdict } = editing;
  const node = storage.find((candidate) => candidate.id === place.nodeId);
  return (
    <>
      <NodeSelect nodeId={place.nodeId} onNodeChange={setNodeId} />
      {node !== undefined && (
        <FolderField
          label="Folder"
          nodeId={node.id}
          nodeName={node.name}
          check={check}
          confirmVerb="Use"
          allowNewFolder
        />
      )}
      {verdict?.usable === true && verdict.network && (
        <Details summary="This folder is on a network share.">
          Files placed here are copied when a hard link isn&apos;t possible, so imports into it can
          be slower.
        </Details>
      )}
    </>
  );
}

/** A root's Node and folder fields between the dialog's own fields, then its actions. */
export function PlaceForm({
  editing,
  onSubmit,
  lead,
  children,
}: {
  editing: Editing;
  onSubmit: () => void;
  lead?: ReactNode;
  children: ReactNode;
}) {
  const { form } = editing;
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      <form.AppForm>
        {lead}
        <form.AppField name="path">{() => <PlaceFields editing={editing} />}</form.AppField>
        {children}
      </form.AppForm>
    </form>
  );
}

/** A save that refuses with a sentence, and whether it is running. */
export function useSaving<A extends unknown[]>(
  action: (...args: A) => Promise<void>,
  refuse: (error: unknown) => string | undefined,
) {
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const run = async (...args: A) => {
    setBusy(true);
    setFailure(undefined);
    try {
      await action(...args);
    } catch (error) {
      setFailure(refuse(error));
    } finally {
      setBusy(false);
    }
  };
  return { busy, failure, run };
}

export function PlaceActions({
  busy,
  failure,
  onCancel,
  children,
}: {
  busy: boolean;
  failure: string | undefined;
  onCancel: () => void;
  children: ReactNode;
}) {
  return (
    <>
      {failure !== undefined && <AlertLine text={failure} />}
      <DialogActions>
        <Button disabled={busy} onClick={onCancel}>
          Cancel
        </Button>
        {children}
      </DialogActions>
    </>
  );
}
