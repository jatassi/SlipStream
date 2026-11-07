import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { DownloadclientsExternalClient as External } from "../api/gen/model";
import { deleteDownloadClient } from "../api/gen/query/downloadclients/downloadclients";
import { countWords } from "../format";
import { dropExternalClient } from "../live/adopt";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";

const torrents = { one: "torrent", many: "torrents" } as const;

function heldWords({ name, torrents: held }: External): string {
  return held === 0
    ? `SlipStream stops checking ${name}. Nothing in ${name} changes.`
    : `${name} holds ${countWords(held, torrents)} SlipStream tracks. They stay in ${name}, and SlipStream stops tracking them.`;
}

/** Delete an external client. The client and its torrents stay as they are. */
function DeleteClientDialog({
  client,
  open,
  onOpenChange,
}: {
  client: External;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  return (
    <ConfirmDialog
      title={`Delete ${client.name}?`}
      confirmLabel={client.torrents === 0 ? "Delete" : "Stop tracking and delete"}
      look="fail"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={async () => {
        try {
          await deleteDownloadClient(client.id, client.torrents === 0 ? {} : { stopTracking: true });
        } catch (error) {
          throw new Error(sentenceOf(error), { cause: error });
        }
        dropExternalClient(queryClient, client.id);
        toast(`Deleted ${client.name}`);
      }}
    >
      <p>{heldWords(client)}</p>
    </ConfirmDialog>
  );
}

export function DeleteClientButton({ client }: { client: External }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        look="ghost"
        size="small"
        onClick={() => {
          setOpen(true);
        }}
      >
        Delete
      </Button>
      <DeleteClientDialog client={client} open={open} onOpenChange={setOpen} />
    </>
  );
}
