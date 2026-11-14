import { useQueryClient } from "@tanstack/react-query";

import type { SettingField as Field } from "../api/gen/model";
import {
  updateMediaServerSettings,
  useGetMediaServerSettingsSuspense,
} from "../api/gen/query/mediaservers/mediaservers";
import { durationWords } from "../format";
import { adoptMediaServerSettings } from "../live/adopt-media-servers";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

const keys = {
  wait: "mediaServers.waitBeforeSending",
  sessions: "mediaServers.sessionsEvery",
  renew: "tasks.plex-sign-in.cadence",
} as const;

function words(settings: readonly Field[], key: string): string {
  const field = settings.find((each) => each.key === key);
  if (field?.kind !== "duration") {
    throw new Error(`the Media Servers settings have no duration ${key}`);
  }
  return durationWords(field.value);
}

/** The six settings that apply to every Media Server, led by what they add up to. */
export function MediaServerSettings() {
  const queryClient = useQueryClient();
  const { settings } = useGetMediaServerSettingsSuspense().data;
  return (
    <Section title="Sending and sessions">
      <Details
        fields
        summary={`SlipStream sends a folder's changes ${words(settings, keys.wait)} after its last change, checks what's playing every ${words(settings, keys.sessions)} and renews the Plex sign-in every ${words(settings, keys.renew)}. The defaults suit most servers.`}
      >
        <div className="mt-2 w-full">
          <SettingGroup>
            {settings.map((field) => (
              <SettingField
                key={field.key}
                field={field}
                onSave={async (value) => {
                  adoptMediaServerSettings(
                    queryClient,
                    await updateMediaServerSettings({ values: { [field.key]: value } }),
                  );
                }}
              />
            ))}
          </SettingGroup>
        </div>
      </Details>
    </Section>
  );
}
