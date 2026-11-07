import { Link, createFileRoute, notFound } from "@tanstack/react-router";

import {
  getGetDownloadClientSettingsSuspenseQueryOptions,
  getListDownloadClientsSuspenseQueryOptions,
  useListDownloadClientsSuspense,
} from "../../../../api/gen/query/downloadclients/downloadclients";
import { getListNodesSuspenseQueryOptions } from "../../../../api/gen/query/nodes/nodes";
import { getListRootsSuspenseQueryOptions } from "../../../../api/gen/query/roots/roots";
import { getGetNodeConfigSuspenseQueryOptions } from "../../../../api/gen/query/system/system";
import { getListContentTypesSuspenseQueryOptions } from "../../../../api/gen/query/works/works";
import { ensureQuery } from "../../../../api/query-client";
import { ClientVerdict } from "../../../../download-clients/client-verdict";
import { ClientLimits } from "../../../../download-clients/limits";
import { sectionOf } from "../../../../shell/node-sections";
// oxlint-disable-next-line import/max-dependencies -- the loader reads the client's, the Node's and the folder field's queries
import { Skeleton } from "../../../../ui/skeleton";
import { nodeOf, useNode } from "../-node";
import { NodeProblem } from "../-node-problem";
import { TorrentConnection } from "../-torrent-connection";

const section = sectionOf("torrent-client");

export const Route = createFileRoute("/_app/nodes/$nodeId/torrent-client")({
  staticData: { title: section.label },
  loader: async ({ context: { queryClient }, params: { nodeId } }) => {
    const { nodes } = await ensureQuery(queryClient, getListNodesSuspenseQueryOptions());
    const node = nodeOf(nodes, nodeId);
    if (section.shows?.(node, nodes) === false) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
      throw notFound();
    }
    await Promise.all([
      ensureQuery(queryClient, getListDownloadClientsSuspenseQueryOptions()),
      ensureQuery(queryClient, getGetDownloadClientSettingsSuspenseQueryOptions(nodeId)),
      ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      node.standing === "in-step" &&
        ensureQuery(queryClient, getGetNodeConfigSuspenseQueryOptions(nodeId)),
    ]);
  },
  pendingComponent: () => <Skeleton label="Loading the torrent client" />,
  errorComponent: NodeProblem,
  component: function TorrentClient() {
    const node = useNode(Route.useParams().nodeId);
    return (
      <div className="max-w-[640px]">
        <Verdict nodeId={node.id} />
        <TorrentConnection node={node} online={node.standing === "in-step"} />
        <ClientLimits nodeId={node.id} />
        <p className="mt-6 text-ink-3">
          <Link
            to="/activity/torrents"
            search={{ node: node.id, tab: "all" }}
            className="text-ink-2 underline hover:text-ink-1"
          >
            Torrents
          </Link>{" "}
          shows what {node.name} downloads and seeds.
        </p>
      </div>
    );
  },
});

function Verdict({ nodeId }: { nodeId: string }) {
  const card = useListDownloadClientsSuspense().data.cards.find((one) => one.nodeId === nodeId);
  return (
    card !== undefined && (
      <div className="mb-6">
        <ClientVerdict card={card} />
      </div>
    )
  );
}
