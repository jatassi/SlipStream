import { Tick02Icon } from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import type { NodesNode } from "../../../api/gen/model";
import { Chevron } from "../../../ui/chevron";
import { Icon } from "../../../ui/icon";
import { List, ListRow } from "../../../ui/list";
import { Select } from "../../../ui/select";
import { Sheet, SheetCancel, SheetGroup, SheetItem } from "../../../ui/sheet";
import { type NodeState, useNodeState } from "./-node";

const dots: { readonly [S in NodeState]: { word: string; color: string } } = {
  online: { word: "online", color: "bg-ok" },
  offline: { word: "offline", color: "bg-ink-4" },
  mismatched: { word: "on another version", color: "bg-warn" },
  restarting: { word: "restarting", color: "bg-ink-1 shadow-live-glow" },
};

function StateDot({ node }: { node: NodesNode }) {
  const dot = dots[useNodeState(node)];
  return (
    <span className="flex shrink-0 items-center">
      <span aria-hidden="true" className={`size-1.5 rounded-capsule ${dot.color}`} />
      <span className="sr-only">{dot.word}</span>
    </span>
  );
}

// Switching Node keeps the section, so a reader on Logs stays on Logs.
function useSwitchNode(): (nodeId: string) => void {
  const navigate = useNavigate();
  return (nodeId) => {
    void navigate({ to: ".", params: (params: object) => ({ ...params, nodeId }) });
  };
}

export function NodePicker({ nodes, current }: { nodes: readonly NodesNode[]; current: string }) {
  const switchTo = useSwitchNode();
  return (
    <div className="mb-6 w-72 max-w-full">
      <Select
        label="Node"
        options={nodes.map((node) => ({
          value: node.id,
          label: node.name,
          leading: <StateDot node={node} />,
        }))}
        value={current}
        onValueChange={switchTo}
      />
    </div>
  );
}

export function PhoneNodePicker({
  nodes,
  current,
}: {
  nodes: readonly NodesNode[];
  current: NodesNode;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="mb-7">
      <List label="Node">
        <ListRow
          title={current.name}
          meta="Node"
          leading={<StateDot node={current} />}
          trailing={<Chevron />}
          render={
            // oxlint-disable-next-line jsx-a11y/control-has-associated-label -- ListRow fills the button with the Node's name
            <button
              type="button"
              onClick={() => {
                setOpen(true);
              }}
            />
          }
        />
      </List>
      <Sheet title="Node" open={open} onOpenChange={setOpen}>
        <NodeChoices nodes={nodes} current={current.id} />
        <SheetCancel />
      </Sheet>
    </div>
  );
}

function NodeChoices({ nodes, current }: { nodes: readonly NodesNode[]; current: string }) {
  const switchTo = useSwitchNode();
  return (
    <SheetGroup>
      {nodes.map((node) => (
        <SheetItem
          key={node.id}
          trailing={node.id === current ? <Icon icon={Tick02Icon} className="size-5" /> : undefined}
          onClick={() => {
            switchTo(node.id);
          }}
        >
          <span className="flex items-center gap-3">
            <StateDot node={node} />
            {node.name}
          </span>
        </SheetItem>
      ))}
    </SheetGroup>
  );
}
