import { RefreshIcon } from "@hugeicons/core-free-icons";

import type { NodesNode } from "../../../api/gen/model";
import { askRestart } from "../../../shell/restart/restart-state";
import { Button } from "../../../ui/button";
import { Spinner } from "../../../ui/spinner";
import { AddNode } from "./-add-node/add-node";

function RestartButton({ node, restarting }: { node: NodesNode; restarting: boolean }) {
  return (
    <Button
      icon={RefreshIcon}
      disabled={restarting}
      onClick={() => {
        askRestart(node);
      }}
    >
      Restart
    </Button>
  );
}

export function NodeRestarting({ name }: { name: string }) {
  return (
    <p className="flex items-center gap-2 text-(length:--body-size) text-ink-1">
      <Spinner />
      <output>{name} is restarting…</output>
    </p>
  );
}

export function NodeActions({ node, restarting }: { node: NodesNode; restarting: boolean }) {
  return (
    <div className="flex gap-2">
      <AddNode />
      <RestartButton node={node} restarting={restarting} />
    </div>
  );
}
