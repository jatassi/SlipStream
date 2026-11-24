import { useQueryClient } from "@tanstack/react-query";

import { patchMoveSettings, useGetMoveSettingsSuspense } from "../api/gen/query/moves/moves";
import { adoptMoveSettings } from "../live/adopt-move-settings";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

/** Settings > Library roots' "Moves" heading: when a waiting switch reaches Needs you, how long the old files wait for Plex, and how long a torrent waits for its destination's check. */
export function MovesSettings() {
  const { settings } = useGetMoveSettingsSuspense().data;
  const queryClient = useQueryClient();
  const save = (key: string) => async (value: unknown) => {
    adoptMoveSettings(queryClient, await patchMoveSettings({ values: { [key]: value } }));
  };
  return (
    <Section title="Moves">
      <div className="flex flex-col gap-5">
        <p className="text-ink-2">
          A moved Work switches to its new root once nobody is playing it and no job has its files
          open. SlipStream deletes the old files once Plex has found the new ones, or after the time
          below.
        </p>
        <SettingGroup>
          {settings.map((field) => (
            <SettingField key={field.key} field={field} onSave={save(field.key)} />
          ))}
        </SettingGroup>
      </div>
    </Section>
  );
}
