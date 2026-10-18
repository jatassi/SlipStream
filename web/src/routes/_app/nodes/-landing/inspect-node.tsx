import { Link } from "@tanstack/react-router";

import { Button } from "../../../../ui/button";
import { Chip } from "../../../../ui/chip";
import { InspectorSection } from "../../../../ui/inspector";
import { Findings } from "../-findings";
import type { Acting } from "../-manage/dialogs";
import { StateDot } from "../-state-dot";
import type { NodeRow } from "./view";

/** A Node in the Inspector: its state, its Roles, and what can be done to it. */
export function NodeWords({ node, act }: { node: NodeRow; act: (acting: Acting) => void }) {
  const on = (kind: Acting["kind"]) => () => {
    act({ kind, nodeId: node.id });
  };
  return (
    <>
      <InspectorSection>
        <p className="flex items-center gap-2">
          <StateDot state={node.state} />
          {node.plaque}
        </p>
        <p className="mt-2 text-ink-3">{node.facts.slice(1).join(" · ")}</p>
        <Findings findings={node.findings} label={`What ${node.name} found`} />
      </InspectorSection>
      <InspectorSection title="Roles">
        {node.roles.length === 0 ? (
          <p className="text-ink-3">No Roles yet</p>
        ) : (
          <span className="flex flex-wrap gap-1.5">
            {node.roles.map(({ role, words }) => (
              <Chip key={role} look="outline">
                {words}
              </Chip>
            ))}
          </span>
        )}
      </InspectorSection>
      <div className="flex flex-wrap gap-2">
        <Button
          look="primary"
          nativeButton={false}
          render={<Link to="/nodes/$nodeId" params={{ nodeId: node.id }} />}
        >
          Open {node.name}
        </Button>
        <Button onClick={on("rename")}>Rename</Button>
        {node.holds && <Button onClick={on("write-off")}>Write off</Button>}
        <Button look="ghost" onClick={on("remove")}>
          Remove
        </Button>
      </div>
    </>
  );
}
