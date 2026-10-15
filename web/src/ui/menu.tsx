import { Menu as BaseMenu } from "@base-ui/react/menu";
import { Tick02Icon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { Fragment, type ReactElement } from "react";

import { Icon } from "./icon";
import { usePortalDensity } from "./portal-density";
import { Sheet, SheetCancel, SheetGroup, SheetItem, SheetToggle, tones, type Tone } from "./sheet";
import { useShell } from "./shell-context";

export type MenuItem =
  | {
      kind?: "action";
      label: string;
      icon?: IconSvgElement;
      tone?: Tone;
      onSelect: () => void;
    }
  | {
      kind: "toggle";
      label: string;
      icon?: IconSvgElement;
      checked: boolean;
      onCheckedChange: (checked: boolean) => void;
    };

export type MenuEntry = MenuItem | "separator";

export type MenuProps = {
  trigger: ReactElement;
  label: string;
  items: readonly MenuEntry[];
};

function groupsOf(items: readonly MenuEntry[]): MenuItem[][] {
  const groups: MenuItem[][] = [[]];
  for (const entry of items) {
    if (entry === "separator") {
      groups.push([]);
    } else {
      groups.at(-1)?.push(entry);
    }
  }
  return groups.filter((group) => group.length > 0);
}

const keyOf = (group: readonly MenuItem[]) => group.map((item) => item.label).join("\n");

export function Menu(props: MenuProps) {
  return useShell() === "phone" ? <ActionSheet {...props} /> : <PopupMenu {...props} />;
}

function ActionSheet({ trigger, label, items }: MenuProps) {
  return (
    <Sheet trigger={trigger} label={label}>
      {groupsOf(items).map((group) => (
        <SheetGroup key={keyOf(group)}>
          {group.map((item) =>
            item.kind === "toggle" ? (
              <SheetToggle
                key={item.label}
                icon={item.icon}
                checked={item.checked}
                onCheckedChange={item.onCheckedChange}
              >
                {item.label}
              </SheetToggle>
            ) : (
              <SheetItem key={item.label} icon={item.icon} tone={item.tone} onClick={item.onSelect}>
                {item.label}
              </SheetItem>
            ),
          )}
        </SheetGroup>
      ))}
      <SheetCancel />
    </Sheet>
  );
}

const popup =
  "max-h-(--available-height) min-w-[220px] origin-(--transform-origin) overflow-y-auto rounded-card bg-popover p-1 text-ink-1 shadow-float outline-none backdrop-blur-[30px] backdrop-saturate-[1.7] transition-[translate,scale,opacity] duration-(--duration-menu) ease-standard data-starting-style:opacity-0 data-ending-style:opacity-0 motion-safe:data-starting-style:translate-y-2 motion-safe:data-starting-style:scale-[.985] motion-safe:data-ending-style:translate-y-2 motion-safe:data-ending-style:scale-[.985]";

const item =
  "flex h-(--menu-item-height) cursor-default items-center gap-2 rounded-item px-3 text-(length:--body-size) leading-(--body-line-height) outline-none select-none data-highlighted:bg-fill-hover focus-visible:shadow-[inset_0_-2px_0_var(--color-glow)]";

function PopupMenu({ trigger, label, items }: MenuProps) {
  const { density, probe } = usePortalDensity();
  return (
    <>
      {probe}
      <BaseMenu.Root>
        <BaseMenu.Trigger render={trigger} />
        <BaseMenu.Portal data-density={density}>
          <BaseMenu.Positioner align="start" sideOffset={6} collisionPadding={12}>
            <BaseMenu.Popup aria-label={label} className={popup}>
              {groupsOf(items).map((group, index) => (
                <Fragment key={keyOf(group)}>
                  {index > 0 && <BaseMenu.Separator className="mx-2 my-1 h-px bg-hairline" />}
                  {group.map((entry) =>
                    entry.kind === "toggle" ? (
                      <BaseMenu.CheckboxItem
                        key={entry.label}
                        checked={entry.checked}
                        onCheckedChange={(checked) => {
                          entry.onCheckedChange(checked);
                        }}
                        className={`${item} ${tones.default}`}
                      >
                        {entry.icon && <Icon icon={entry.icon} className="size-4 shrink-0" />}
                        <span className="flex-1">{entry.label}</span>
                        <BaseMenu.CheckboxItemIndicator className="text-glow">
                          <Icon icon={Tick02Icon} className="size-4" />
                        </BaseMenu.CheckboxItemIndicator>
                      </BaseMenu.CheckboxItem>
                    ) : (
                      <BaseMenu.Item
                        key={entry.label}
                        onClick={entry.onSelect}
                        className={`${item} ${tones[entry.tone ?? "default"]}`}
                      >
                        {entry.icon && <Icon icon={entry.icon} className="size-4 shrink-0" />}
                        {entry.label}
                      </BaseMenu.Item>
                    ),
                  )}
                </Fragment>
              ))}
            </BaseMenu.Popup>
          </BaseMenu.Positioner>
        </BaseMenu.Portal>
      </BaseMenu.Root>
    </>
  );
}
