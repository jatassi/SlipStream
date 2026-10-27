import { useQueryClient } from "@tanstack/react-query";
import { z } from "zod";

import { DialogActions } from "../account/dialog-actions";
import type { IndexersTracker } from "../api/gen/model";
import { renameTracker } from "../api/gen/query/indexers/indexers";
import { adoptTracker } from "../live/adopt";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { submitErrors, useAppForm } from "../ui/form";
import { toast } from "../ui/toast";

type Props = { tracker: IndexersTracker; onClose: () => void };

const renameForm = z.object({
  name: z.string().trim().min(1, "Give the Tracker a name").max(100, "Use 100 characters or fewer"),
});

function RenameForm({ tracker, onClose }: Props) {
  const queryClient = useQueryClient();
  const form = useAppForm({
    defaultValues: { name: tracker.name },
    validators: { onSubmit: renameForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        const saved = await renameTracker(tracker.id, renameForm.parse(value));
        adoptTracker(queryClient, saved);
        toast(`Renamed ${tracker.name} to ${saved.name}`);
        onClose();
      } catch (error) {
        formApi.setErrorMap({ onSubmit: submitErrors(error, { tracker_name_taken: "name" }) });
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
        <p className="text-ink-2">
          Its Indexers, Grabs and torrents stay with it, and Releases from it keep reaching it.
        </p>
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

export function RenameTrackerDialog({ tracker, onClose }: Props) {
  return (
    <Dialog
      title={`Rename ${tracker.name}`}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <RenameForm tracker={tracker} onClose={onClose} />
    </Dialog>
  );
}
