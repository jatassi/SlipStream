import { useRouter } from "@tanstack/react-router";

import { updateRootSettings, useGetRootSettingsSuspense } from "../api/gen/query/roots/roots";
import { durationWords } from "../format";
import { Details } from "../ui/details";
import { SettingField, SettingGroup } from "../ui/setting-field";

export function SampleEvery() {
  const { sampleEvery } = useGetRootSettingsSuspense().data;
  const router = useRouter();
  const save = async (value: number | null) => {
    await updateRootSettings({ sampleEvery: value });
    await router.invalidate();
  };
  return (
    <Details
      summary={`Storage Nodes check each volume every ${durationWords(sampleEvery.value)}, and again after each import.`}
    >
      <div className="mt-2 w-full max-w-[560px]">
        <SettingGroup>
          <SettingField field={sampleEvery} onSave={save} />
        </SettingGroup>
      </div>
    </Details>
  );
}
