import { z } from "zod";

import { renameNode } from "../../../../api/gen/query/nodes/nodes";
import { DialogActions } from "../../../../account/dialog-actions";
import { Button } from "../../../../ui/button";
import { Dialog, DialogClose } from "../../../../ui/dialog";
import { submitErrors, useAppForm } from "../../../../ui/form";

const renameForm = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Give the Node a name")
    .max(63, "A Node's name is at most 63 characters"),
});

function RenameForm({
  nodeId,
  name,
  onRenamed,
}: {
  nodeId: string;
  name: string;
  onRenamed: () => Promise<void>;
}) {
  const form = useAppForm({
    defaultValues: { name },
    validators: { onSubmit: renameForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        await renameNode(nodeId, renameForm.parse(value));
        await onRenamed();
      } catch (error) {
        formApi.setErrorMap({
          onSubmit: submitErrors(error, { node_name_taken: "name", node_name_invalid: "name" }),
        });
      }
    },
  });
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
        <form.AppField name="name">
          {(field) => <field.TextField label="Name" autoComplete="off" spellCheck={false} />}
        </form.AppField>
        <form.FormAlert />
        <DialogActions>
          <DialogClose>
            <Button>Cancel</Button>
          </DialogClose>
          <form.SubmitButton label="Rename" />
        </DialogActions>
      </form.AppForm>
    </form>
  );
}

/** Rename a Node: history and its own status page take the new name. */
export function RenameDialog({
  nodeId,
  name,
  onClose,
  onRenamed,
}: {
  nodeId: string;
  name: string;
  onClose: () => void;
  onRenamed: () => Promise<void>;
}) {
  return (
    <Dialog
      title={`Rename ${name}`}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <RenameForm nodeId={nodeId} name={name} onRenamed={onRenamed} />
    </Dialog>
  );
}
