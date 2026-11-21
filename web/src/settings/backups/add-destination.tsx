import { useState } from "react";

import type { NodesNode } from "../../api/gen/model";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { AddFields } from "./add-destination-fields";

export function AddDestinationButton({ nodes }: { nodes: readonly NodesNode[] }) {
  const [open, setOpen] = useState(false);
  return (
    <Dialog
      title="Add a Backup destination"
      open={open}
      onOpenChange={setOpen}
      trigger={<Button>Add a Backup destination</Button>}
    >
      {open && (
        <AddFields
          nodes={nodes}
          close={() => {
            setOpen(false);
          }}
        />
      )}
    </Dialog>
  );
}
