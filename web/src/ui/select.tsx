import { Select as BaseSelect } from "@base-ui/react/select";
import { Tick02Icon, UnfoldMoreIcon } from "@hugeicons/core-free-icons";

import { Icon } from "./icon";

export type SelectOption<V extends string> = { value: V; label: string };

export function Select<V extends string>({
  label,
  options,
  value,
  onValueChange,
}: {
  label: string;
  options: readonly SelectOption<V>[];
  value: V;
  onValueChange: (value: V) => void;
}) {
  return (
    <BaseSelect.Root
      items={options}
      value={value}
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
        <BaseSelect.Trigger className="flex h-(--input-height) cursor-default items-center justify-between gap-3 rounded-item bg-well pr-2 pl-3 text-body text-ink-1 ring-1 ring-hairline ring-inset select-none in-data-[density=touch]:text-input-touch">
          <BaseSelect.Value />
          <BaseSelect.Icon className="flex text-ink-3">
            <Icon icon={UnfoldMoreIcon} className="size-4" />
          </BaseSelect.Icon>
        </BaseSelect.Trigger>
      </div>
      <SelectPopup options={options} />
    </BaseSelect.Root>
  );
}

function SelectPopup<V extends string>({ options }: { options: readonly SelectOption<V>[] }) {
  return (
    <BaseSelect.Portal>
      <BaseSelect.Positioner
        alignItemWithTrigger={false}
        sideOffset={6}
        collisionPadding={12}
        className="z-50 outline-none"
      >
        <BaseSelect.Popup className="min-w-(--anchor-width) origin-(--transform-origin) rounded-card bg-popover p-1 text-ink-1 shadow-float backdrop-blur-[30px] backdrop-saturate-[1.7] transition-[opacity,translate,scale] duration-(--duration-menu) ease-standard outline-none data-ending-style:opacity-0 data-starting-style:translate-y-2 data-starting-style:scale-[.985] data-starting-style:opacity-0 motion-reduce:data-starting-style:translate-y-0 motion-reduce:data-starting-style:scale-100">
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
                <BaseSelect.ItemText>{option.label}</BaseSelect.ItemText>
              </BaseSelect.Item>
            ))}
          </BaseSelect.List>
        </BaseSelect.Popup>
      </BaseSelect.Positioner>
    </BaseSelect.Portal>
  );
}
