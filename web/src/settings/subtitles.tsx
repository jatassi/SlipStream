import {
  getGetSubtitleSettingsSuspenseQueryOptions,
  getListSubtitleProvidersSuspenseQueryOptions,
  useGetSubtitleSettingsSuspense,
  useListSubtitleProvidersSuspense,
} from "../api/gen/query/subtitles/subtitles";
import {
  getListContentTypesSuspenseQueryOptions,
  useListContentTypesSuspense,
} from "../api/gen/query/works/works";
import { ensureQuery, freshQuery } from "../api/query-client";
import { Section } from "../ui/section";
import type { SettingsSection } from "./section";
import { useKeyDialog } from "./subtitle-key-dialog";
import { Lead, ProfileRows } from "./subtitle-lead";
import { NodeCards, loadNodeSettings } from "./subtitle-nodes";
import { SubtitleProviders } from "./subtitle-providers";
import { SubtitleSearch } from "./subtitle-search";

/** The lead and its next step, the Profiles that want subtitles, the providers, Subtitle search, then the Storage Nodes' scans. */
function SubtitlesSettings() {
  const settings = useGetSubtitleSettingsSuspense().data;
  const { providers } = useListSubtitleProvidersSuspense().data;
  const types = useListContentTypesSuspense().data.contentTypes.filter((type) => type.enabled);
  const { openKey, dialog } = useKeyDialog();
  return (
    <>
      <Section title="Subtitles">
        <div className="flex flex-col gap-5">
          <Lead settings={settings} providers={providers} types={types} openKey={openKey} />
          {settings.profiles.length > 0 && <ProfileRows profiles={settings.profiles} />}
        </div>
      </Section>
      <SubtitleProviders types={types} openKey={openKey} />
      <SubtitleSearch settings={settings.settings} />
      <NodeCards />
      {dialog}
    </>
  );
}

export const subtitlesSection: SettingsSection = {
  id: "subtitles",
  label: "Subtitles",
  load: async (queryClient) => {
    await Promise.all([
      freshQuery(queryClient, getGetSubtitleSettingsSuspenseQueryOptions()),
      freshQuery(queryClient, getListSubtitleProvidersSuspenseQueryOptions()),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      loadNodeSettings(queryClient),
    ]);
  },
  component: SubtitlesSettings,
  health: ["subtitles"],
};
