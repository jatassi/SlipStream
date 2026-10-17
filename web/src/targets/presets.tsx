import { ArrowDown01Icon } from "@hugeicons/core-free-icons";

import { useListQualitiesSuspense } from "../api/gen/query/targets/targets";
import { Button } from "../ui/button";
import { Menu } from "../ui/menu";
import type { TargetDraft } from "./draft";
import type { SectionProps } from "./editor";
import { placeCutoff, presetOrder, rankedOf } from "./order";

/** Refills the order from a preset, keeping what is accepted; No upgrades moves the cutoff to the bottom. */
export function PresetMenu({ draft, onChange }: SectionProps<TargetDraft>) {
  const { presets } = useListQualitiesSuspense().data;
  return (
    <Menu
      label="Presets"
      trigger={
        <Button size="small" icon={ArrowDown01Icon}>
          Fill from a preset
        </Button>
      }
      items={[
        ...presets.map((preset) => ({
          label: preset.name,
          onSelect: () => {
            onChange({ ...draft, order: presetOrder(preset.tiers, rankedOf(draft.order)) });
          },
        })),
        "separator",
        {
          label: "No upgrades",
          onSelect: () => {
            onChange({ ...draft, order: placeCutoff(draft.order, "bottom") });
          },
        },
      ]}
    />
  );
}
