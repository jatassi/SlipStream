import { Drawer } from "@base-ui/react/drawer";
import type { IconSvgElement } from "@hugeicons/react";
import type { ReactElement, ReactNode } from "react";

import { Icon } from "./icon";
import { SwitchControl } from "./switch";

type Naming = { title: string; label?: never } | { label: string; title?: never };

export type SheetProps = Naming & {
  /** Stands at its full height, for content that fills it, such as a list with a bar under it. */
  fill?: boolean;
  trigger?: ReactElement;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  children: ReactNode;
};

const scrim =
  "fixed inset-0 bg-[rgb(0_0_0/0.5)] opacity-[calc(1-var(--drawer-swipe-progress))] transition-opacity duration-(--duration-sheet-in) ease-spring data-swiping:duration-0 data-starting-style:opacity-0 data-ending-style:opacity-0 data-ending-style:duration-(--duration-sheet-out)";

const heights = {
  content: "max-h-[calc(100dvh-env(safe-area-inset-top)-12px)]",
  full: "h-[calc(100dvh-env(safe-area-inset-top)-12px)]",
} as const;

const popup =
  "flex w-full flex-col overflow-y-auto overscroll-contain rounded-t-sheet bg-sheet px-4 pt-2 pb-[max(44px,calc(env(safe-area-inset-bottom)+16px))] text-ink-1 outline-none backdrop-blur-[30px] backdrop-saturate-[1.7] [transform:translateY(var(--drawer-swipe-movement-y))] transition-[translate,transform,opacity] duration-(--duration-sheet-in) ease-spring data-swiping:duration-0 data-swiping:select-none data-ending-style:duration-(--duration-sheet-out) motion-safe:data-starting-style:translate-y-full motion-safe:data-ending-style:translate-y-full motion-reduce:data-starting-style:opacity-0 motion-reduce:data-ending-style:opacity-0";

const row =
  "flex h-(--menu-item-height) w-full items-center gap-3 px-4 text-left text-sentence-touch outline-none hover:bg-fill-hover focus-visible:bg-fill-hover focus-visible:shadow-[inset_0_-2px_0_var(--color-glow)]";

export const tones = { default: "text-ink-1", danger: "text-fail-ink" } as const;

export type Tone = keyof typeof tones;

// The sheet only exists in the phone shell, whose density is always Touch.
export function Sheet({ fill = false, trigger, title, label, children, ...root }: SheetProps) {
  return (
    <Drawer.Root {...root}>
      {trigger && <Drawer.Trigger render={trigger} />}
      <Drawer.Portal data-density="touch">
        <Drawer.Backdrop className={scrim} />
        <Drawer.Viewport className="fixed inset-0 flex items-end justify-center">
          <Drawer.Popup
            aria-label={label}
            aria-modal="true"
            className={`${popup} ${heights[fill ? "full" : "content"]}`}
          >
            <span
              aria-hidden="true"
              className="mx-auto mb-2 block h-[5px] w-[38px] shrink-0 rounded-capsule bg-ink-4/60"
            />
            {title !== undefined && (
              <Drawer.Title className="px-1 pt-1 pb-3 text-title">{title}</Drawer.Title>
            )}
            <Drawer.Content className={`flex flex-col gap-2 ${fill ? "min-h-0 flex-1" : ""}`}>
              {children}
            </Drawer.Content>
          </Drawer.Popup>
        </Drawer.Viewport>
      </Drawer.Portal>
    </Drawer.Root>
  );
}

function RowIcon({ icon }: { icon: IconSvgElement | undefined }) {
  return icon ? (
    <Icon icon={icon} className="row-icon shrink-0" />
  ) : (
    <span aria-hidden="true" className="hidden size-5 shrink-0 group-has-[.row-icon]/rows:block" />
  );
}

export function SheetGroup({ children }: { children: ReactNode }) {
  return (
    <div className="group/rows flex flex-col divide-y divide-hairline overflow-hidden rounded-card bg-fill">
      {children}
    </div>
  );
}

export function SheetItem({
  icon,
  tone = "default",
  disabled = false,
  trailing,
  onClick,
  children,
}: {
  icon?: IconSvgElement | undefined;
  tone?: Tone | undefined;
  disabled?: boolean | undefined;
  trailing?: ReactNode;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Drawer.Close
      disabled={disabled}
      onClick={onClick}
      className={`${row} ${tones[tone]} disabled:opacity-40`}
    >
      <RowIcon icon={icon} />
      <span className="flex-1">{children}</span>
      {trailing}
    </Drawer.Close>
  );
}

export function SheetToggle({
  icon,
  checked,
  onCheckedChange,
  children,
}: {
  icon?: IconSvgElement | undefined;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  children: ReactNode;
}) {
  return (
    <label className={`${row} ${tones.default} has-focus-visible:bg-fill-hover`}>
      <RowIcon icon={icon} />
      <span className="flex-1">{children}</span>
      <SwitchControl checked={checked} onCheckedChange={onCheckedChange} />
    </label>
  );
}

export function SheetCancel() {
  return (
    <Drawer.Close className="flex h-(--menu-item-height) w-full shrink-0 items-center justify-center rounded-card bg-fill text-sentence-touch text-ink-1 hover:bg-fill-hover">
      Cancel
    </Drawer.Close>
  );
}
