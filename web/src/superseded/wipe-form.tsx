import { useState } from "react";
import { z } from "zod";

import { isApiError } from "../api/errors";
import { wipeSupersededController } from "../api/gen/query/superseded/superseded";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { useAppForm } from "../ui/form";

const joinCodeHelp = "Create a Join Code on the new Controller, then enter it here.";

const wipeForm = z.object({
  joinCode: z.string().trim().min(1, joinCodeHelp),
});

function WipeConfirm({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<void>;
}) {
  return (
    <ConfirmDialog
      title="Wipe this Controller?"
      confirmLabel="Wipe and join as a Worker"
      look="fail"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={onConfirm}
    >
      <p>
        SlipStream backs up its library to its Backups folder, deletes the library database and
        restarts as a Worker of the new Controller. Media files and torrents stay on this machine.
        This Node seeds them again once it joins the new Controller.
      </p>
    </ConfirmDialog>
  );
}

/** The page's recommendation: wipe this Controller and join the new one with its Join Code. */
export function WipeForm({ onWiping }: { onWiping: () => void }) {
  const [asking, setAsking] = useState(false);
  const form = useAppForm({
    defaultValues: { joinCode: "" },
    validators: { onSubmit: wipeForm },
    onSubmit: () => {
      setAsking(true);
    },
  });
  const wipe = async () => {
    try {
      await wipeSupersededController({ joinCode: wipeForm.parse(form.state.values).joinCode });
      onWiping();
    } catch (error) {
      if (!isApiError(error) || error.code !== "join_code_damaged") {
        throw error;
      }
      form.setErrorMap({ onSubmit: { fields: { joinCode: error.title } } });
    }
  };
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
        <form.AppField name="joinCode">
          {(field) => (
            <field.TextField
              label="Join Code"
              description={joinCodeHelp}
              data
              autoCapitalize="characters"
              autoComplete="off"
              spellCheck={false}
            />
          )}
        </form.AppField>
        <div>
          <form.SubmitButton label="Wipe and join as a Worker" />
        </div>
      </form.AppForm>
      <WipeConfirm open={asking} onOpenChange={setAsking} onConfirm={wipe} />
    </form>
  );
}
