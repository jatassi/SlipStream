import {
  getGetMediaServerSettingsSuspenseQueryOptions,
  getListMediaServerKindsSuspenseQueryOptions,
  getListMediaServersSuspenseQueryOptions,
  useListMediaServerKindsSuspense,
  useListMediaServersSuspense,
} from "../api/gen/query/mediaservers/mediaservers";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../api/gen/query/nodes/nodes";
import { ensureQuery, freshQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { Section } from "../ui/section";
import { ServerList } from "./server-row";
import { AddServerButton } from "./server-sheet";
import { PlexAccount, loadPlexAccount } from "./plex-account";
import { MediaServerSettings } from "./settings-panel";

function MediaServersSection() {
  const { servers } = useListMediaServersSuspense().data;
  const { kinds } = useListMediaServerKindsSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  return (
    <>
      <Section
        title="Media Servers"
        count={servers.length}
        action={<AddServerButton kinds={kinds} nodes={nodes} />}
      >
        <ServerList servers={servers} kinds={kinds} />
      </Section>
      <PlexAccount servers={servers} kinds={kinds} />
      <MediaServerSettings />
    </>
  );
}

export const mediaServersSection: SettingsSection = {
  id: "media-servers",
  label: "Media Servers",
  load: async (queryClient) => {
    await Promise.all([
      freshQuery(queryClient, getListMediaServersSuspenseQueryOptions()),
      ensureQuery(queryClient, getListMediaServerKindsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
      freshQuery(queryClient, getGetMediaServerSettingsSuspenseQueryOptions()),
      loadPlexAccount(queryClient),
    ]);
  },
  component: MediaServersSection,
};
