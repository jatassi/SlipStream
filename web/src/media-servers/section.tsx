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
import { Empty } from "../shell/empty";
import { List } from "../ui/list";
import { Section } from "../ui/section";
import { ServerRow } from "./server-row";
import { AddServerButton } from "./server-sheet";
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
        {servers.length === 0 ? (
          <Empty>
            {
              "No Media Servers yet. Add a Plex, Jellyfin or Emby server, and SlipStream sends it every change to your library's files."
            }
          </Empty>
        ) : (
          <List label="Media Servers">
            {servers.map((server) => (
              <ServerRow key={server.id} server={server} kinds={kinds} />
            ))}
          </List>
        )}
      </Section>
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
    ]);
  },
  component: MediaServersSection,
  health: ["media-servers"],
};
