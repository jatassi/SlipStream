import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import type { MediaserversServerView as Server } from "../../api/gen/model";
import { removeMediaServer } from "../../api/gen/query/mediaservers/mediaservers";
import { dropMediaServer } from "../../live/adopt-media-servers";
import { Button } from "../../ui/button";
import { ConfirmDialog } from "../../ui/confirm-dialog";
import { toast } from "../../ui/toast";

/** Removes the server and leaves for the Media Servers section. */
export function RemoveServer({ server }: { server: Server }) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setOpen(true);
        }}
      >
        Remove
      </Button>
      <ConfirmDialog
        title={`Remove ${server.name}?`}
        confirmLabel="Remove"
        look="fail"
        open={open}
        onOpenChange={setOpen}
        onConfirm={async () => {
          await removeMediaServer(server.id);
          dropMediaServer(queryClient, server.id);
          toast(`Removed ${server.name}`);
          await navigate({ to: "/settings/$section", params: { section: "media-servers" } });
        }}
      >
        <p>
          {`Removing it also deletes its waiting changes and its play history. Nothing changes on ${server.name} itself.`}
        </p>
      </ConfirmDialog>
    </>
  );
}
