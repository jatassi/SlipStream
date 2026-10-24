import { useRouter } from "@tanstack/react-router";

import type { SettingField as Field } from "../api/gen/model";
import {
  getGetMetadataSettingsSuspenseQueryOptions,
  updateMetadataSettings,
  useGetMetadataSettingsSuspense,
} from "../api/gen/query/metadata/metadata";
import { freshQuery } from "../api/query-client";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { ClearCache } from "./metadata-cache";
import type { SettingsSection } from "./section";

const chosen = new Set(["metadata.language", "metadata.region"]);

const optionLabel = (field: Field | undefined) =>
  field?.kind === "choice"
    ? (field.options.find((option) => option.value === field.value)?.label ?? field.value)
    : "";

function MetadataSettings() {
  const { settings } = useGetMetadataSettingsSuspense().data;
  const router = useRouter();
  const field = (setting: Field) => (
    <SettingField
      key={setting.key}
      field={setting}
      onSave={async (value) => {
        await updateMetadataSettings({ values: { [setting.key]: value } });
        await router.invalidate();
      }}
    />
  );
  const language = settings.find((setting) => setting.key === "metadata.language");
  const region = settings.find((setting) => setting.key === "metadata.region");
  return (
    <Section title="Metadata">
      <Details
        summary={`Titles and overviews read in ${optionLabel(language)}, and release dates and certifications come from ${optionLabel(region)}. Pick the language you read and the country you watch in; the rest suits most libraries.`}
        action={
          <div className="my-2 w-full">
            <SettingGroup>
              {settings
                .filter((setting) => chosen.has(setting.key))
                .map((setting) => field(setting))}
            </SettingGroup>
          </div>
        }
      >
        <div className="mt-2 flex w-full flex-col gap-4">
          <SettingGroup>
            {settings
              .filter((setting) => !chosen.has(setting.key))
              .map((setting) => field(setting))}
          </SettingGroup>
          <ClearCache />
        </div>
      </Details>
    </Section>
  );
}

export const metadataSection: SettingsSection = {
  id: "metadata",
  label: "Metadata",
  load: async (queryClient) => {
    await freshQuery(queryClient, getGetMetadataSettingsSuspenseQueryOptions());
  },
  component: MetadataSettings,
};
