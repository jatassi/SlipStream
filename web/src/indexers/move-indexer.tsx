import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { IndexersIndexer, NodesNode } from "../api/gen/model";
import { updateIndexer } from "../api/gen/query/indexers/indexers";
import { adoptIndexer } from "../live/adopt";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Select } from "../ui/select";
import { toast } from "../ui/toast";
import { nodeName } from "./words";

type DialogProps = {
  indexer: IndexersIndexer;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

const refused = (error: unknown) => new Error(sentenceOf(error), { cause: error });

/** The Nodes an Indexer may search from: every Node assigned Searching. */
export const searchingNodes = (nodes: readonly NodesNode[]) =>
  nodes.filter((node) =>
    node.roles.some((view) => view.role === "searching" && view.state !== "covering"),
  );

export function MoveIndexerDialog({
  indexer,
  nodes,
  open,
  onOpenChange,
}: DialogProps & { nodes: readonly NodesNode[] }) {
  const queryClient = useQueryClient();
  const options = searchingNodes(nodes).map((node) => ({ value: node.id, label: node.name }));
  const [to, setTo] = useState(indexer.homeNodeId);
  const moving = to !== indexer.homeNodeId;
  return (
    <ConfirmDialog
      title={`Move ${indexer.name}`}
      confirmLabel="Move"
      confirmDisabled={!moving}
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={async () => {
        try {
          adoptIndexer(queryClient, await updateIndexer(indexer.id, { homeNodeId: to }));
        } catch (error) {
          throw refused(error);
        }
        toast(`${indexer.name} searches from ${nodeName(nodes, to)} now`);
      }}
    >
      <p>
        An Indexer searches, signs in and fetches from its home Node. Moving it tests it from the
        new one first.
      </p>
      <Select label="Search from" options={options} value={to} onValueChange={setTo} />
      {moving && indexer.signsIn && (
        <p className="text-warn">
          {indexer.name} will see a sign-in from {nodeName(nodes, to)}&apos;s address. It may ask
          for a captcha or email you about it.
        </p>
      )}
    </ConfirmDialog>
  );
}
