import { Link, Outlet, useMatch } from "@tanstack/react-router";

import { useListNodesSuspense } from "../../../api/gen/query/nodes/nodes";
import { Page } from "../../../shell/page";
import { PageTabs, pageTab } from "../../../ui/page-tabs";
import { useShell } from "../../../ui/shell-context";
import { nodeOf, useNodeState } from "./-node";
import { NodePicker, PhoneNodePicker } from "./-node-picker";
import { NodeRestarting, RestartButton } from "./-restart";
import { SectionList } from "./-section-list";
import { nodeSections } from "./-sections";

// On phone each section is its own page: the Node's top page carries the
// picker and lists the others, and the shell titles a pushed section and
// links back to the Node.
export function NodeLayout({ nodeId }: { nodeId: string }) {
  const { nodes } = useListNodesSuspense().data;
  const node = nodeOf(nodes, nodeId);
  const restarting = useNodeState(node) === "restarting";
  const body = restarting ? <NodeRestarting name={node.name} /> : <Outlet />;
  const restart = <RestartButton node={node} restarting={restarting} />;
  const top = useMatch({ from: "/_app/nodes/$nodeId/", shouldThrow: false }) !== undefined;
  if (useShell() === "phone") {
    return (
      <Page {...(top && { action: restart })}>
        {top && nodes.length > 1 && <PhoneNodePicker nodes={nodes} current={node} />}
        {body}
        {top && <SectionList node={node} />}
      </Page>
    );
  }
  return (
    <Page title={node.name} action={restart}>
      {nodes.length > 1 && <NodePicker nodes={nodes} current={node.id} />}
      <PageTabs label={node.name}>
        {nodeSections.map((section) => (
          <Link
            key={section.id}
            to={section.to}
            params={{ nodeId }}
            activeOptions={{ exact: true }}
            className={pageTab}
          >
            {section.label}
          </Link>
        ))}
      </PageTabs>
      {body}
    </Page>
  );
}
