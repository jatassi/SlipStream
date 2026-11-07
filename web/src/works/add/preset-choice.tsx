import type { WorksContentType, WorksPresetCounts } from "../../api/gen/model";
import { countWords } from "../../format";
import { Select } from "../../ui/select";
import { type Preset, lowered } from "../monitoring/labels";
import type { AddChoices, MonitorControl } from "./choices";
import { monitoringOf } from "./sentence";

/** The monitoring choice, each preset with what it monitors now while the preview counts it. */
export function PresetChoice({
  type,
  control,
  counts,
  choices,
  onChange,
}: {
  type: WorksContentType;
  control: Extract<MonitorControl, { kind: "choice" }>;
  counts: WorksPresetCounts | null;
  choices: AddChoices;
  onChange: (preset: Preset) => void;
}) {
  const items = lowered(type.item);
  const options = control.options.map(({ preset, label }) => ({
    value: preset,
    label:
      counts === null
        ? label
        : `${label} · ${countWords(monitoringOf(counts, { ...choices, preset }).monitors, items)}`,
  }));
  return (
    <Select
      label={control.label}
      options={options}
      value={choices.preset}
      onValueChange={onChange}
    />
  );
}
