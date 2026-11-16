import type { TargetsTargetPropers } from "../api/gen/model";
import { RadioGroup } from "../ui/radio";
import type { TargetDraft } from "./draft";
import type { SectionProps } from "./editor";

const choices: readonly { value: TargetsTargetPropers; label: string; description: string }[] = [
  {
    value: "prefer_and_upgrade",
    label: "Prefer and upgrade",
    description:
      "A proper or repack ranks above other releases of its quality and replaces a file of that quality, even past the cutoff. A repack must come from the file's release group.",
  },
  {
    value: "prefer_no_upgrade",
    label: "Prefer, don't upgrade",
    description:
      "A proper or repack ranks above other releases of its quality. Its revision alone doesn't replace a file you have.",
  },
  {
    value: "dont_prefer",
    label: "Don't prefer",
    description: "Propers and repacks rank like any other release of their quality.",
  },
];

export function PropersSection({ draft, onChange }: SectionProps<TargetDraft>) {
  return (
    <div className="flex flex-col gap-3">
      <p className="text-ink-2">
        Most Targets keep Prefer and upgrade. Choose Don&apos;t prefer when Custom Formats score
        propers and repacks, as TRaSH Guides do.
      </p>
      <RadioGroup
        label="Propers and repacks"
        options={choices}
        value={draft.propers}
        onValueChange={(propers) => {
          onChange({ ...draft, propers });
        }}
      />
    </div>
  );
}
