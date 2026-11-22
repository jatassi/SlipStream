import { useState } from "react";

import type {
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
} from "../api/gen/model";
import { ClientTorrentsDialog } from "../moves/client-move-dialog";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import type { NodeChoice } from "./client-fields";
import { ClientSheet } from "./client-sheet";

type Chosen = "node" | "torrents";

function Chooser({
  name,
  open,
  onOpenChange,
  onChoose,
}: {
  name: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onChoose: (chosen: Chosen) => void;
}) {
  return (
    <Dialog title={`Move ${name}`} open={open} onOpenChange={onOpenChange}>
      <div className="flex flex-col gap-4">
        <p>
          If this client now runs on another Node, change its Node. Nothing is copied. To copy its
          torrents into another client, move them.
        </p>
        <div className="flex flex-wrap justify-end gap-2">
          <Button
            onClick={() => {
              onChoose("torrents");
            }}
          >
            Move its torrents
          </Button>
          <Button
            look="primary"
            onClick={() => {
              onChoose("node");
            }}
          >
            Change its Node
          </Button>
        </div>
      </div>
    </Dialog>
  );
}

/**
 * "Move client…": leads with the Node change, which copies nothing, and
 * offers to move the client's torrents into another client instead.
 */
export function MoveClientButton({
  client,
  kinds,
  nodes,
}: {
  client: External;
  kinds: readonly Kind[];
  nodes: readonly NodeChoice[];
}) {
  const [choosing, setChoosing] = useState(false);
  const [chosen, setChosen] = useState<Chosen>();
  const choose = (next: Chosen) => {
    setChoosing(false);
    setChosen(next);
  };
  const close = (open: boolean) => {
    if (!open) setChosen(undefined);
  };
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setChoosing(true);
        }}
      >
        Move client…
      </Button>
      <Chooser name={client.name} open={choosing} onOpenChange={setChoosing} onChoose={choose} />
      {chosen === "node" && (
        <ClientSheet
          opened={{ kind: "edit", client, changingNode: true }}
          kinds={kinds}
          nodes={nodes}
          open
          onOpenChange={close}
        />
      )}
      {chosen === "torrents" && (
        <ClientTorrentsDialog
          name={client.name}
          from={{ nodeId: client.nodeId, clientId: client.id }}
          open
          onOpenChange={close}
        />
      )}
    </>
  );
}
