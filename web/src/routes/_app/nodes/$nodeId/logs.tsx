import { createFileRoute } from "@tanstack/react-router";

import {
  getGetNodeConfigSuspenseQueryOptions,
  useGetNodeConfigSuspense,
} from "../../../../api/gen/query/system/system";
import { ensureQuery } from "../../../../api/query-client";
import { sectionOf } from "../../../../shell/node-sections";
import { Skeleton } from "../../../../ui/skeleton";
import { settingOf } from "../../../../nodes/node-setting";
import { LogViewer } from "../-logs/log-viewer";
import { NodeProblem } from "../-node-problem";

const section = sectionOf("logs");

export const Route = createFileRoute("/_app/nodes/$nodeId/logs")({
  staticData: { title: section.label },
  loader: ({ context, params }) =>
    ensureQuery(context.queryClient, getGetNodeConfigSuspenseQueryOptions(params.nodeId)),
  pendingComponent: () => <Skeleton label="Loading the log" />,
  errorComponent: NodeProblem,
  component: function Logs() {
    const { nodeId } = Route.useParams();
    const config = useGetNodeConfigSuspense(nodeId).data;
    const cap = settingOf(config, "log.viewer_lines", "count").running;
    return <LogViewer key={nodeId} nodeId={nodeId} cap={cap} />;
  },
});
