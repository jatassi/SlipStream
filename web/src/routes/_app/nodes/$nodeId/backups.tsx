import { Link, createFileRoute } from "@tanstack/react-router";

import { getGetNodeConfigSuspenseQueryOptions } from "../../../../api/gen/query/system/system";
import { ensureQuery } from "../../../../api/query-client";
import { sectionOf } from "../../../../shell/node-sections";
import { Skeleton } from "../../../../ui/skeleton";
import { NodeSettings } from "../-config";
import { NodeProblem } from "../-node-problem";

const section = sectionOf("backups");

export const Route = createFileRoute("/_app/nodes/$nodeId/backups")({
  staticData: { title: section.label },
  loader: ({ context, params }) =>
    ensureQuery(context.queryClient, getGetNodeConfigSuspenseQueryOptions(params.nodeId)),
  pendingComponent: () => <Skeleton label="Loading the settings" />,
  errorComponent: NodeProblem,
  component: function NodeBackups() {
    const { nodeId } = Route.useParams();
    return (
      <div className="flex max-w-[640px] flex-col gap-4">
        <p className="text-ink-2">
          SlipStream keeps every Backup in this folder, and copies them from here to each Backup
          destination. The schedule, the Backup destinations and the Backup passphrase are in{" "}
          <Link
            to="/settings/$section"
            params={{ section: "backups" }}
            className="underline decoration-ink-3 underline-offset-4 hover:decoration-ink-1"
          >
            {"Settings > Backups"}
          </Link>
          .
        </p>
        <NodeSettings nodeId={nodeId} section="backups" />
      </div>
    );
  },
});
