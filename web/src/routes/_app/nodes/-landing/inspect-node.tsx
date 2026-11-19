import { Link } from "@tanstack/react-router";

import { RoleChips } from "../../../../nodes/role-chips";
import { NodeStateMark } from "../../../../nodes/state-mark";
import { Button } from "../../../../ui/button";
import { InspectorSection } from "../../../../ui/inspector";
import { Findings } from "../-findings";
import type { Acting } from "../-manage/dialogs";
import type { NodeRow } from "./view";

/** A Node in the Inspector: its state, its Roles, and what can be done to it. */
export function NodeWords({ node, act }: { node: NodeRow; act: (acting: Acting) => void }) {
  const on = (kind: Acting["kind"]) => () => {
    act({ kind, nodeId: node.id });
  };
  return (
    <>
      <InspectorSection>
        <NodeStateMark state={node.state} />
        <p className="mt-2 text-ink-3">{node.facts.slice(1).join(" · ")}</p>
        <Findings findings={node.findings} label={`What ${node.name} found`} />
      </InspectorSection>
      <InspectorSection title="Roles">
        <RoleChips roles={node.roles} />
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
