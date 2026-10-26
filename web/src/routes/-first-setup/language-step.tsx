import { useState } from "react";

import type { SettingChoiceField, SettingField } from "../../api/gen/model";
import {
  updateMetadataSettings,
  useGetMetadataSettingsSuspense,
} from "../../api/gen/query/metadata/metadata";
import { useAttempt } from "../../targets/attempt";
import { Button } from "../../ui/button";
import { Select } from "../../ui/select";
import { StepFooter } from "./frame";
import { suggestPlace } from "./language";
import type { StepBodyProps } from "./screens";

function choiceOf(settings: readonly SettingField[], key: string): SettingChoiceField {
  const field = settings.find((setting) => setting.key === key);
  if (field?.kind !== "choice") {
    throw new Error(`getMetadataSettings has no choice ${key}`);
  }
  return field;
}

// The browser's suggestion fills a field only while it holds the default, so
// a value the Admin set stays.
function usePrefilled(language: SettingChoiceField, region: SettingChoiceField) {
  return useState(() => {
    const suggested = suggestPlace(
      navigator.languages,
      language.options.map((option) => option.value),
      region.options.map((option) => option.value),
    );
    return {
      language: language.isDefault ? (suggested.language ?? language.value) : language.value,
      region: region.isDefault ? (suggested.region ?? region.value) : region.value,
    };
  });
}

export function LanguageStep({ nav }: StepBodyProps) {
  const { settings } = useGetMetadataSettingsSuspense().data;
  const language = choiceOf(settings, "metadata.language");
  const region = choiceOf(settings, "metadata.region");
  const [chosen, setChosen] = usePrefilled(language, region);
  const save = useAttempt(async () => {
    await updateMetadataSettings({
      values: { [language.key]: chosen.language, [region.key]: chosen.region },
    });
    await nav.complete();
  });
  return (
    <>
      <Select
        label={language.label}
        options={language.options}
        value={chosen.language}
        onValueChange={(value) => {
          setChosen({ ...chosen, language: value });
        }}
      />
      <Select
        label={region.label}
        options={region.options}
        value={chosen.region}
        onValueChange={(value) => {
          setChosen({ ...chosen, region: value });
        }}
      />
      <p>File names keep the English title whatever language you pick.</p>
      {save.failure}
      <StepFooter
        nav={nav}
        primary={
          <Button look="primary" busy={save.busy} onClick={save.run}>
            Continue
          </Button>
        }
      />
    </>
  );
}
