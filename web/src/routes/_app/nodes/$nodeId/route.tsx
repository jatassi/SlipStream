import { Link, Outlet, createFileRoute } from "@tanstack/react-router";

import { useListNodesSuspense } from "../../../../api/gen/query/nodes/nodes";
import { Page } from "../../../../shell/page";
import { loaderTitle } from "../../../../shell/titles";
import { PageTabs, pageTab } from "../../../../ui/page-tabs";
import { useShell } from "../../../../ui/shell-context";
import { Skeleton } from "../../../../ui/skeleton";
import { loadNode, nodeOf } from "../-node";
import { NodePicker } from "../-node-picker";
import { nodeSections } from "../-sections";

export const Route = createFileRoute("/_app/nodes/$nodeId")({
  staticData: { title: loaderTitle },
  loader: ({ context, params }) => loadNode(context.queryClient, params.nodeId),
  pendingComponent: () => <Skeleton label="Loading the Node" />,
  component: NodeLayout,
});

// On phone each section is its own page: the Overview lists the others, and
// the shell titles a pushed section and links back to the Node.
function NodeLayout() {
  const { nodeId } = Route.useParams();
  const { nodes } = useListNodesSuspense().data;
  const node = nodeOf(nodes, nodeId);
  if (useShell() === "phone") {
    return (
      <Page>
        <Outlet />
      </Page>
    );
  }
  return (
    <Page title={node.name}>
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
      <Outlet />
    </Page>
  );
}
