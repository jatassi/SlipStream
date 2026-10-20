import { Link } from "@tanstack/react-router";

import {
  getListDownloadClientsSuspenseQueryOptions,
  useListDownloadClientsSuspense,
} from "../api/gen/query/downloadclients/downloadclients";
import { freshQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { Empty } from "../shell/empty";
import { Section } from "../ui/section";
import { ClientCard } from "./card";
import { Thresholds } from "./thresholds";

function DownloadClientsSection() {
  const { cards, thresholds } = useListDownloadClientsSuspense().data;
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
              <ClientCard key={card.nodeId} card={card} />
            ))}
          </div>
        )}
      </Section>
      <Thresholds thresholds={thresholds} />
    </>
  );
}

export const downloadClientsSection: SettingsSection = {
  id: "download-clients",
  label: "Download Clients",
  load: async (queryClient) => {
    await freshQuery(queryClient, getListDownloadClientsSuspenseQueryOptions());
  },
  component: DownloadClientsSection,
};
