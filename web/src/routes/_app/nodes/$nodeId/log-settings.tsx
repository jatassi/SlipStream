import { createFileRoute } from "@tanstack/react-router";

import {
  getGetNodeConfigSuspenseQueryOptions,
  getGetNodeSystemSuspenseQueryOptions,
  useGetNodeSystemSuspense,
} from "../../../../api/gen/query/system/system";
import { ensureQuery } from "../../../../api/query-client";
import { sectionOf } from "../../../../shell/node-sections";
import { ReadOnlyField } from "../../../../ui/read-only-field";
import { SettingGroup } from "../../../../ui/setting-field";
import { Skeleton } from "../../../../ui/skeleton";
import { NodeSettings } from "../-config";
import { DownloadLogs } from "../-download";
import { NodeProblem } from "../-node-problem";

const section = sectionOf("log-settings");

export const Route = createFileRoute("/_app/nodes/$nodeId/log-settings")({
  staticData: { title: section.label },
  loader: ({ context, params }) =>
    Promise.all([
      ensureQuery(context.queryClient, getGetNodeConfigSuspenseQueryOptions(params.nodeId)),
      ensureQuery(context.queryClient, getGetNodeSystemSuspenseQueryOptions(params.nodeId)),
    ]),
  pendingComponent: () => <Skeleton label="Loading the settings" />,
  errorComponent: NodeProblem,
  component: function LogSettings() {
    const { nodeId } = Route.useParams();
    const { logsDir } = useGetNodeSystemSuspense(nodeId).data;
    return (
      <div className="flex max-w-[640px] flex-col gap-4">
        <NodeSettings nodeId={nodeId} section="log" />
        <SettingGroup>
          <div className="flex flex-col items-start gap-3">
            <div className="self-stretch">
              <ReadOnlyField label="Log folder" value={logsDir} />
            </div>
            <DownloadLogs nodeId={nodeId} />
          </div>
        </SettingGroup>
      </div>
    );
  },
});
