import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import { deleteNotificationConnection } from "../api/gen/query/notifications/notifications";
import { dropConnection } from "../live/adopt-notifications";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";

/** Deletes a connection after asking. Its waiting messages are dropped. */
export function DeleteConnectionButton({ label, id }: { label: string; id: string }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        look="ghost"
        size="small"
        aria-label={`Delete ${label}`}
        onClick={() => {
          setOpen(true);
        }}
      >
        Delete
      </Button>
      <ConfirmDialog
        title={`Delete ${label}?`}
        confirmLabel="Delete"
        look="fail"
        open={open}
        onOpenChange={setOpen}
        onConfirm={async () => {
          try {
            await deleteNotificationConnection(id);
          } catch (error) {
            throw new Error(sentenceOf(error), { cause: error });
          }
          dropConnection(queryClient, id);
          toast(`Deleted ${label}`);
        }}
      >
        <p>{"Its waiting messages won't be sent."}</p>
      </ConfirmDialog>
    </>
  );
}
