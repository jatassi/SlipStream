import { Radio } from "@base-ui/react/radio";
import { RadioGroup } from "@base-ui/react/radio-group";

import { useShell } from "./shell-context";

export type SegmentedOption<V extends string> = { value: V; label: string; count?: number };

const tracks = {
  desktop: "inline-flex",
  phone: "grid w-full auto-cols-[minmax(max-content,1fr)] grid-flow-col overflow-x-auto",
} as const;

export function Segmented<V extends string>({
  label,
  options,
  value,
  onValueChange,
}: {
  label: string;
  options: readonly SegmentedOption<V>[];
  value: V;
  onValueChange: (value: V) => void;
}) {
  return (
    <RadioGroup
      aria-label={label}
      value={value}
      onValueChange={(next) => {
        onValueChange(next);
      }}
      className={`${tracks[useShell()]} gap-0.5 rounded-capsule bg-fill p-0.5 ring-1 ring-hairline ring-inset`}
    >
      {options.map((option) => (
        <Radio.Root
          key={option.value}
          value={option.value}
          className="inline-flex h-7 cursor-default items-center justify-center gap-1.5 rounded-capsule px-3 text-body font-medium whitespace-nowrap text-ink-2 transition-[background-color,color] duration-(--duration-control) ease-standard select-none in-data-[density=touch]:h-11 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-medium data-checked:bg-selected data-checked:text-ink-1 data-checked:shadow-[0_1px_3px_rgba(0,0,0,.28)] motion-reduce:transition-none"
        >
          {option.label}
          {option.count === undefined ? null : (
            <span className="text-ink-3 tabular-nums">{option.count}</span>
          )}
        </Radio.Root>
      ))}
    </RadioGroup>
  );
}
