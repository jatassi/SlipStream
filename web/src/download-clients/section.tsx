import { Link } from "@tanstack/react-router";

import {
  getListDownloadClientsSuspenseQueryOptions,
  useListDownloadClientsSuspense,
} from "../api/gen/query/downloadclients/downloadclients";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../api/gen/query/nodes/nodes";
import { ensureQuery, freshQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { Empty } from "../shell/empty";
import { Section } from "../ui/section";
import { NodeCard } from "./node-card";
import { Thresholds } from "./thresholds";
import { Unattached } from "./unattached";

function DownloadClientsSection() {
  const { cards, orphans, kinds, thresholds } = useListDownloadClientsSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const downloading = cards.map((card) => ({ id: card.nodeId, name: card.nodeName }));
  return (
    <>
      <Section title="Clients">
        {cards.length === 0 ? (
          <Empty>
            No Node downloads yet. Give one Downloading in its{" "}
            <Link to="/nodes" className="text-ink-1 underline">
              Roles
            </Link>{" "}
            and its built-in client shows here.
          </Empty>
        ) : (
          <div className="flex flex-col gap-3">
            {cards.map((card) => (
              <NodeCard key={card.nodeId} card={card} kinds={kinds} nodes={downloading} />
            ))}
          </div>
        )}
      </Section>
      {orphans.length > 0 && <Unattached orphans={orphans} kinds={kinds} nodes={nodes} />}
      <Thresholds thresholds={thresholds} />
    </>
  );
}

export const downloadClientsSection: SettingsSection = {
  id: "download-clients",
  label: "Download Clients",
  load: async (queryClient) => {
    await Promise.all([
      freshQuery(queryClient, getListDownloadClientsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ]);
  },
  component: DownloadClientsSection,
  health: ["download-clients"],
};
