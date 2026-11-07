import { Switch as BaseSwitch } from "@base-ui/react/switch";

/** On, off, or mixed when some of what the switch covers is on. */
export type SwitchChecked = boolean | "mixed";

export const switchTrack =
  "flex h-5 w-8.5 shrink-0 items-center rounded-capsule bg-[hsl(var(--h)_20%_90%/.22)] p-0.5 transition-[background-color] duration-(--duration-chevron) ease-standard in-data-[density=touch]:h-7.75 in-data-[density=touch]:w-12.75 data-checked:bg-[hsl(var(--h)_62%_46%)] data-mixed:bg-[hsl(var(--h)_62%_46%/.55)] motion-reduce:transition-none";

export const switchKnob =
  "size-4 rounded-capsule bg-solid shadow-[0_1px_3px_rgba(0,0,0,.3)] transition-[translate] duration-280 ease-standard in-data-[density=touch]:size-6.75 data-checked:translate-x-3.5 in-data-[density=touch]:data-checked:translate-x-5 data-mixed:translate-x-1.75 in-data-[density=touch]:data-mixed:translate-x-2.5 motion-reduce:transition-none";

export function mixedAttributes(checked: SwitchChecked) {
  return checked === "mixed" ? ({ "aria-checked": "mixed", "data-mixed": "" } as const) : {};
}

export function Switch({
  label,
  checked,
  onCheckedChange,
  disabled = false,
}: {
  label: string;
  checked: SwitchChecked;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <label className="inline-flex min-h-(--control-height) cursor-default items-center gap-3 rounded-capsule text-body text-ink-1 select-none has-data-disabled:opacity-40 in-data-[density=touch]:text-body-touch">
      {label}
      <SwitchControl checked={checked} onCheckedChange={onCheckedChange} disabled={disabled} />
    </label>
  );
}

/**
 * The track and knob alone, for a row whose own label makes the whole row the
 * target, or that names itself with label. A mixed switch turns on.
 */
export function SwitchControl({
  label,
  checked,
  onCheckedChange,
  disabled = false,
}: {
  label?: string;
  checked: SwitchChecked;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
}) {
  const mixed = mixedAttributes(checked);
  return (
    <BaseSwitch.Root
      aria-label={label}
      checked={checked === true}
      onCheckedChange={(next) => {
        onCheckedChange(next);
      }}
      disabled={disabled}
      {...mixed}
      className={switchTrack}
    >
      <BaseSwitch.Thumb {...mixed} className={switchKnob} />
    </BaseSwitch.Root>
  );
}
