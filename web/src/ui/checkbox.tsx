import { Checkbox as BaseCheckbox } from "@base-ui/react/checkbox";
import { MinusSignIcon, Tick02Icon } from "@hugeicons/core-free-icons";

import { Icon } from "./icon";

/** How the change was made: shift held means a range, as a list reads it. */
export type CheckedBy = { readonly shiftKey: boolean };

export function Checkbox({
  label,
  hideLabel = false,
  checked,
  indeterminate = false,
  onCheckedChange,
  disabled = false,
}: {
  label: string;
  /** The label names it for assistive tech only, as in a list's pick column. */
  hideLabel?: boolean;
  checked: boolean;
  indeterminate?: boolean;
  onCheckedChange: (checked: boolean, by: CheckedBy) => void;
  disabled?: boolean;
}) {
  return (
    <label className="inline-flex min-h-(--control-height) cursor-default items-center gap-2.5 text-body text-ink-1 select-none has-data-disabled:opacity-40 in-data-[density=touch]:gap-3 in-data-[density=touch]:text-body-touch">
      <BaseCheckbox.Root
        {...(hideLabel && { "aria-label": label })}
        checked={checked}
        indeterminate={indeterminate}
        onCheckedChange={(next, { event }) => {
          onCheckedChange(next, { shiftKey: event instanceof MouseEvent && event.shiftKey });
        }}
        disabled={disabled}
        className="flex size-4 shrink-0 items-center justify-center rounded-[4px] bg-well text-ink-1 ring-1 ring-hairline-strong ring-inset transition-[background-color] duration-(--duration-control) ease-standard in-data-[density=touch]:size-5.5 in-data-[density=touch]:rounded-[6px] data-checked:bg-[hsl(var(--h)_62%_46%)] data-checked:ring-0 data-indeterminate:bg-[hsl(var(--h)_62%_46%)] data-indeterminate:ring-0 motion-reduce:transition-none"
      >
        <BaseCheckbox.Indicator>
          <Icon
            icon={indeterminate ? MinusSignIcon : Tick02Icon}
            className="size-3 in-data-[density=touch]:size-4"
          />
        </BaseCheckbox.Indicator>
      </BaseCheckbox.Root>
      {!hideLabel && label}
    </label>
  );
}
