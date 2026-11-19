import { Outlet, useMatch } from "@tanstack/react-router";

import { useListNodesSuspense } from "../../../api/gen/query/nodes/nodes";
import { useNodeState } from "../../../nodes/state";
import { Page } from "../../../shell/page";
import { useShell } from "../../../ui/shell-context";
import { nodeOf } from "./-node";
import { NodePicker, PhoneNodePicker } from "./-node-picker";
import { NodeActions, NodeRestarting } from "./-actions";
import { SectionList, SectionTabs } from "./-section-list";

// On phone each section is its own page: the Node's top page carries the
// picker and lists the others, and the shell titles a pushed section and
// links back to the Node. A restarting Worker's section stays mounted under
// the notice, so the Logs keep their lines and draw the restart's divider.
export function NodeLayout({ nodeId }: { nodeId: string }) {
  const { nodes } = useListNodesSuspense().data;
  const node = nodeOf(nodes, nodeId);
  const restarting = useNodeState(node) === "restarting";
  const body = (
    <>
      {restarting && <NodeRestarting name={node.name} />}
      <div className={restarting ? "hidden" : "contents"}>
        <Outlet />
      </div>
    </>
  );
  const actions = <NodeActions node={node} restarting={restarting} />;
  const top = useMatch({ from: "/_app/nodes/$nodeId/", shouldThrow: false }) !== undefined;
  if (useShell() === "phone") {
    return (
      <Page {...(top && { action: actions })}>
        {top && nodes.length > 1 && <PhoneNodePicker nodes={nodes} current={node} />}
        {body}
        {top && <SectionList node={node} nodes={nodes} />}
      </Page>
    );
  }
  return (
    <Page title={node.name} action={actions}>
      {nodes.length > 1 && <NodePicker nodes={nodes} current={node.id} />}
      <SectionTabs node={node} nodes={nodes} />
      {body}
    </Page>
  );
}
