import { useQueryClient } from "@tanstack/react-query";

import type {
  ConversionsNodeConverting,
  ConversionsNodeSettings,
  SettingCountField,
  SettingSwitchField,
  SettingTimes,
  SettingWindowField,
} from "../api/gen/model";
import { updateNodeConverting } from "../api/gen/query/conversions/conversions";
import { adoptNodeConverting } from "../live/conversions";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { toast } from "../ui/toast";

const defaultHours: SettingTimes = { start: "01:00", end: "07:00" };

const shared = { applies: "live", pendingRestart: false, source: "database" } as const;

const settingsOf = ({ atOnce, atOnceDefault, hours, processor }: ConversionsNodeConverting) => ({
  atOnce: atOnce === atOnceDefault ? null : atOnce,
  hours,
  processor,
});

const atOnceOf = ({
  atOnce,
  atOnceDefault,
}: ConversionsNodeConverting): SettingCountField & { kind: "count" } => ({
  ...shared,
  kind: "count",
  key: "atOnce",
  label: "Conversions at once",
  min: 1,
  max: 16,
  unit: { one: "Conversion", many: "Conversions" },
  default: atOnceDefault,
  value: atOnce,
  running: atOnce,
  isDefault: atOnce === atOnceDefault,
});

const switchOf = (
  key: string,
  label: string,
  value: boolean,
  initial: boolean,
): SettingSwitchField & { kind: "switch" } => ({
  ...shared,
  kind: "switch",
  key,
  label,
  default: initial,
  value,
  running: value,
  isDefault: value === initial,
});

const windowOf = (window: SettingTimes, zone: string): SettingWindowField & { kind: "window" } => ({
  ...shared,
  kind: "window",
  key: "hours.window",
  label: "Start Conversions between",
  default: defaultHours,
  value: window,
  running: window,
  isDefault: window.start === defaultHours.start && window.end === defaultHours.end,
  zone,
});

/** Conversions at once, the hours a Node starts Conversions in, and the processor switch, each saved on its own. */
export function ConvertingSettings({
  nodeId,
  converting,
  zone,
}: {
  nodeId: string;
  converting: ConversionsNodeConverting;
  zone: string;
}) {
  const queryClient = useQueryClient();
  const save = async (change: Partial<ConversionsNodeSettings>) => {
    const got = await updateNodeConverting(nodeId, { ...settingsOf(converting), ...change });
    adoptNodeConverting(queryClient, nodeId, got);
    toast("Converting settings saved");
  };
  const { hours, processor } = converting;
  return (
    <SettingGroup>
      <SettingField field={atOnceOf(converting)} onSave={(value) => save({ atOnce: value })} />
      <SettingField
        field={switchOf("hours.on", "Only start Conversions in set hours", hours.on, false)}
        onSave={(value) => save({ hours: { ...hours, on: value ?? false } })}
      />
      {hours.on && (
        <SettingField
          field={windowOf(hours.window, zone)}
          onSave={(value) => save({ hours: { ...hours, window: value ?? defaultHours } })}
        />
      )}
      <SettingField
        field={switchOf("processor", "Use the processor when no GPU can", processor, true)}
        onSave={(value) => save({ processor: value ?? true })}
      />
    </SettingGroup>
  );
}
