import { useState } from "react";

import type { MediaserversServerView as Server } from "../../api/gen/model";
import { refreshMediaServer } from "../../api/gen/query/mediaservers/mediaservers";
import { Button } from "../../ui/button";
import { ConfirmDialog } from "../../ui/confirm-dialog";
import { toast } from "../../ui/toast";

/** Queues a whole library refresh of each library the server's confirmed Path mappings lead into. */
export function RefreshServer({ server }: { server: Server }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setOpen(true);
        }}
      >
        Refresh whole library
      </Button>
      <ConfirmDialog
        title={`Refresh every library on ${server.name}?`}
        confirmLabel="Refresh"
        open={open}
        onOpenChange={setOpen}
        onConfirm={async () => {
          await refreshMediaServer(server.id);
          toast(`Queued a refresh of ${server.name}'s libraries`);
        }}
      >
        <p>
          {`${server.name} scans each library its confirmed Path mappings lead into, which can take a while on a large library. Waiting changes in those libraries count as sent once ${server.name} accepts the refresh.`}
        </p>
      </ConfirmDialog>
    </>
  );
}
