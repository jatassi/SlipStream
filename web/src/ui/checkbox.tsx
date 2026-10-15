import { Checkbox as BaseCheckbox } from "@base-ui/react/checkbox";
import { Tick02Icon } from "@hugeicons/core-free-icons";

import { Icon } from "./icon";

export function Checkbox({
  label,
  checked,
  onCheckedChange,
  disabled = false,
}: {
  label: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <label className="inline-flex min-h-(--control-height) cursor-default items-center gap-2.5 text-body text-ink-1 select-none has-data-disabled:opacity-40 in-data-[density=touch]:gap-3 in-data-[density=touch]:text-body-touch">
      <BaseCheckbox.Root
        checked={checked}
        onCheckedChange={(next) => {
          onCheckedChange(next);
        }}
        disabled={disabled}
        className="flex size-4 shrink-0 items-center justify-center rounded-[4px] bg-well text-ink-1 ring-1 ring-hairline-strong ring-inset transition-[background-color] duration-(--duration-control) ease-standard in-data-[density=touch]:size-5.5 in-data-[density=touch]:rounded-[6px] data-checked:bg-[hsl(var(--h)_62%_46%)] data-checked:ring-0 motion-reduce:transition-none"
      >
        <BaseCheckbox.Indicator>
          <Icon icon={Tick02Icon} className="size-3 in-data-[density=touch]:size-4" />
        </BaseCheckbox.Indicator>
      </BaseCheckbox.Root>
      {label}
    </label>
  );
}
