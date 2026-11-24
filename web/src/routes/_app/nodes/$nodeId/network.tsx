import { createFileRoute } from "@tanstack/react-router";

import {
  getGetNodeNetworkSettingsSuspenseQueryOptions,
  getGetNodeNetworkSuspenseQueryOptions,
} from "../../../../api/gen/query/reachability/reachability";
import { ensureQuery, freshQuery } from "../../../../api/query-client";
import { NetworkSection } from "../../../../remote-access/network-section";
import { sectionOf } from "../../../../shell/node-sections";
import { Skeleton } from "../../../../ui/skeleton";
import { NodeProblem } from "../-node-problem";

const section = sectionOf("network");

export const Route = createFileRoute("/_app/nodes/$nodeId/network")({
  staticData: { title: section.label },
  loader: ({ context, params }) =>
    Promise.all([
      freshQuery(context.queryClient, getGetNodeNetworkSuspenseQueryOptions(params.nodeId)),
      ensureQuery(
        context.queryClient,
        getGetNodeNetworkSettingsSuspenseQueryOptions(params.nodeId),
      ),
    ]),
  pendingComponent: () => <Skeleton label="Loading the network" shape="facts" />,
  errorComponent: NodeProblem,
  component: function Network() {
    return <NetworkSection nodeId={Route.useParams().nodeId} />;
  },
});
