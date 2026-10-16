import { Select as BaseSelect } from "@base-ui/react/select";
import { Tick02Icon, UnfoldMoreIcon } from "@hugeicons/core-free-icons";
import type { ReactNode } from "react";

import { floatingPanel } from "./floating";
import { Icon } from "./icon";
import { usePortalDensity } from "./portal-density";

export type SelectOption<V extends string> = { value: V; label: string; leading?: ReactNode };

export function Select<V extends string>({
  label,
  options,
  value,
  disabled = false,
  onValueChange,
}: {
  label: string;
  options: readonly SelectOption<V>[];
  value: V;
  disabled?: boolean;
  onValueChange: (value: V) => void;
}) {
  const { density, probe } = usePortalDensity();
  const chosen = options.find((option) => option.value === value);
  return (
    <BaseSelect.Root
      items={options}
      value={value}
      disabled={disabled}
      onValueChange={(next) => {
        if (next !== null) {
          onValueChange(next);
        }
      }}
    >
      <div className="flex flex-col gap-1.5">
        <BaseSelect.Label className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
          {label}
        </BaseSelect.Label>
        {probe}
        <BaseSelect.Trigger className="flex h-(--input-height) cursor-default items-center justify-between gap-3 rounded-item bg-well pr-2 pl-3 text-body text-ink-1 ring-1 ring-hairline ring-inset select-none data-disabled:opacity-40 in-data-[density=touch]:text-input-touch">
          <BaseSelect.Value className="flex min-w-0 items-center gap-2">
            {() => (
              <>
                {chosen?.leading}
                <span className="truncate">{chosen?.label}</span>
              </>
            )}
          </BaseSelect.Value>
          <BaseSelect.Icon className="flex text-ink-3">
            <Icon icon={UnfoldMoreIcon} className="size-4" />
          </BaseSelect.Icon>
        </BaseSelect.Trigger>
      </div>
      <SelectPopup options={options} density={density} />
    </BaseSelect.Root>
  );
}

function SelectPopup<V extends string>({
  options,
  density,
}: {
  options: readonly SelectOption<V>[];
  density: string | undefined;
}) {
  return (
    <BaseSelect.Portal data-density={density}>
      <BaseSelect.Positioner
        alignItemWithTrigger={false}
        sideOffset={6}
        collisionPadding={12}
        className="z-50 outline-none"
      >
        <BaseSelect.Popup className={`min-w-(--anchor-width) p-1 ${floatingPanel}`}>
          <BaseSelect.List className="max-h-(--available-height) overflow-y-auto">
            {options.map((option) => (
              <BaseSelect.Item
                key={option.value}
                value={option.value}
                className="flex h-(--menu-item-height) cursor-default items-center gap-2 rounded-item px-3 text-body outline-none select-none in-data-[density=touch]:text-body-touch data-highlighted:bg-fill-hover"
              >
                <span className="flex w-4">
                  <BaseSelect.ItemIndicator>
                    <Icon icon={Tick02Icon} className="size-4" />
                  </BaseSelect.ItemIndicator>
                </span>
                {option.leading}
                <BaseSelect.ItemText>{option.label}</BaseSelect.ItemText>
              </BaseSelect.Item>
            ))}
          </BaseSelect.List>
        </BaseSelect.Popup>
      </BaseSelect.Positioner>
    </BaseSelect.Portal>
  );
}
