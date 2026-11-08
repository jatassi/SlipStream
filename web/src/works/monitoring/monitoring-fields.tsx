import { sentenceOf } from "../../api/errors";
import type { WorksContentType } from "../../api/gen/model";
import { countWords } from "../../format";
import { AlertLine } from "../../ui/alert-line";
import { RadioGroup } from "../../ui/radio";
import { Skeleton } from "../../ui/skeleton";
import { Switch } from "../../ui/switch";
import { type MonitoringChoice, type MonitoringPreview, changeWords, countOf } from "./choice";
import { includeLabel, lowered, monitorNewLabel, presetLabels, presets } from "./labels";

type FieldsProps = {
  readonly type: WorksContentType;
  /** How many Works the choice is for. */
  readonly works: number;
  readonly preview: { readonly data: MonitoringPreview | undefined; readonly error: Error | null };
  readonly choice: MonitoringChoice;
  readonly onChange: (next: MonitoringChoice) => void;
};

function presetOptions(type: WorksContentType, preview: MonitoringPreview, includeAsides: boolean) {
  const items = lowered(type.item);
  return presets.map((preset) => ({
    value: preset,
    label: `${presetLabels[preset](type)} · ${countWords(countOf(preview.counts, preset, includeAsides).monitors, items)}`,
  }));
}

/** The presets with what each monitors, the two rules, and what the choice starts and stops. */
export function MonitoringFields({ type, works, preview, choice, onChange }: FieldsProps) {
  if (preview.data === undefined) {
    return preview.error === null ? (
      <Skeleton label="Counting what each choice monitors" />
    ) : (
      <AlertLine text={sentenceOf(preview.error)} />
    );
  }
  const { includeAsides } = choice;
  return (
    <>
      <RadioGroup
        label={`${type.item.many} to monitor`}
        options={presetOptions(type, preview.data, includeAsides)}
        value={choice.preset}
        onValueChange={(preset) => {
          onChange({ preset, includeAsides, monitorNew: undefined });
        }}
      />
      <div className="flex flex-col">
        {type.aside !== null && (
          <Switch
            label={includeLabel(type.aside)}
            checked={includeAsides}
            onCheckedChange={(on) => {
              onChange({ ...choice, includeAsides: on });
            }}
          />
        )}
        <Switch
          label={monitorNewLabel(type)}
          checked={
            choice.monitorNew ??
            countOf(preview.data.counts, choice.preset, includeAsides).monitorsNew
          }
          onCheckedChange={(on) => {
            onChange({ ...choice, monitorNew: on });
          }}
        />
      </div>
      <p aria-live="polite" className="text-ink-2">
        {changeWords(type, preview.data, works)}
      </p>
    </>
  );
}
