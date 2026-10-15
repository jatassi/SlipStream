import { Autocomplete } from "@base-ui/react/autocomplete";
import { Dialog } from "@base-ui/react/dialog";
import { Search01Icon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { type ReactNode, type RefObject, createContext, use, useRef } from "react";

import { Icon } from "./icon";
import { type ShellKind, useShell } from "./shell-context";

export type PaletteOption = {
  readonly id: string;
  readonly label: string;
  readonly meta?: string | undefined;
  readonly icon: IconSvgElement;
  readonly choose: () => void;
};

export type PaletteGroup = { readonly label: string; readonly items: PaletteOption[] };

const InputRef = createContext<RefObject<HTMLInputElement | null>>({ current: null });

const viewports = {
  desktop: "fixed inset-0 flex items-start justify-center px-4 pt-[12vh]",
  phone: "fixed inset-0",
} as const satisfies Record<ShellKind, string>;

const popups = {
  desktop:
    "flex max-h-[min(600px,80vh)] w-[640px] max-w-full flex-col overflow-hidden rounded-dialog shadow-float-large motion-safe:data-starting-style:scale-[.98] motion-safe:data-ending-style:scale-[.98]",
  phone:
    "flex h-dvh w-full flex-col pt-[env(safe-area-inset-top)] pr-[env(safe-area-inset-right)] pb-[env(safe-area-inset-bottom)] pl-[env(safe-area-inset-left)]",
} as const satisfies Record<ShellKind, string>;

export function CommandPalette({
  label,
  open,
  onOpenChange,
  children,
}: {
  label: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
}) {
  const shell = useShell();
  const input = useRef<HTMLInputElement>(null);
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 bg-[rgb(0_0_0/0.5)] transition-opacity duration-(--duration-palette) ease-standard data-starting-style:opacity-0 data-ending-style:opacity-0" />
        <Dialog.Viewport className={viewports[shell]}>
          <Dialog.Popup
            aria-label={label}
            aria-modal="true"
            initialFocus={input}
            className={`bg-popover text-ink-1 outline-none backdrop-blur-[30px] backdrop-saturate-[1.7] transition-[scale,opacity] duration-(--duration-palette) ease-standard data-starting-style:opacity-0 data-ending-style:opacity-0 ${popups[shell]}`}
          >
            <InputRef value={input}>{children}</InputRef>
          </Dialog.Popup>
        </Dialog.Viewport>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

const options = {
  desktop: "h-11 rounded-item px-3 text-body",
  phone: "h-14 rounded-card px-4 text-body-touch",
} as const satisfies Record<ShellKind, string>;

export function CommandSearch({
  label,
  query,
  onQueryChange,
  groups,
  empty,
}: {
  label: string;
  query: string;
  onQueryChange: (query: string) => void;
  groups: readonly PaletteGroup[];
  empty: ReactNode;
}) {
  const shell = useShell();
  return (
    <Autocomplete.Root
      open
      inline
      mode="none"
      items={groups}
      value={query}
      onValueChange={onQueryChange}
      itemToStringValue={(option: PaletteOption) => option.label}
      autoHighlight="always"
      keepHighlight
    >
      <SearchRow label={label} />
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-2">
        <Autocomplete.Empty>
          <p className="px-3 py-6 text-body text-ink-2">{empty}</p>
        </Autocomplete.Empty>
        <Autocomplete.List className="outline-none">
          {(group: PaletteGroup) => (
            <Autocomplete.Group key={group.label} items={group.items} className="not-first:mt-2">
              <Autocomplete.GroupLabel className="px-3 pt-2 pb-1.5 text-label-caps text-ink-3 uppercase">
                {group.label}
              </Autocomplete.GroupLabel>
              <Autocomplete.Collection>
                {(option: PaletteOption) => <Option key={option.id} option={option} />}
              </Autocomplete.Collection>
            </Autocomplete.Group>
          )}
        </Autocomplete.List>
      </div>
      {shell === "desktop" && <Hints />}
    </Autocomplete.Root>
  );
}

function SearchRow({ label }: { label: string }) {
  const shell = useShell();
  return (
    <div className="flex h-15 shrink-0 items-center gap-3 px-5 text-ink-3 shadow-[inset_0_-1px_0_var(--color-hairline)] transition-[background-color,box-shadow] duration-(--duration-control) focus-within:bg-fill-hover focus-within:shadow-[inset_0_-2px_0_var(--color-glow)]">
      <Icon icon={Search01Icon} className="shrink-0" />
      <Autocomplete.Input
        ref={use(InputRef)}
        aria-label={label}
        placeholder={label}
        className="h-full min-w-0 flex-1 bg-transparent text-input-touch text-ink-1 placeholder:text-ink-3 focus-visible:shadow-none focus-visible:outline-none"
      />
      {shell === "phone" && (
        <Dialog.Close className="shrink-0 rounded-chip text-body-touch font-semibold text-glow">
          Cancel
        </Dialog.Close>
      )}
    </div>
  );
}

function Option({ option }: { option: PaletteOption }) {
  return (
    <Autocomplete.Item
      value={option}
      onClick={option.choose}
      className={`flex cursor-default scroll-my-2 items-center gap-3 text-ink-1 outline-none select-none data-highlighted:bg-fill-hover ${options[useShell()]}`}
    >
      <Icon icon={option.icon} className="size-4.5 shrink-0 text-ink-2" />
      <span className="min-w-0 flex-1 truncate">{option.label}</span>
      {option.meta !== undefined && (
        <span className="shrink-0 text-meta text-ink-3">{option.meta}</span>
      )}
    </Autocomplete.Item>
  );
}

const hints = [
  { keys: "↑↓", label: "Move" },
  { keys: "↵", label: "Open" },
  { keys: "Esc", label: "Close" },
] as const;

function Hints() {
  return (
    <div className="flex h-10 shrink-0 items-center gap-5 px-5 text-meta text-ink-3 shadow-[inset_0_1px_0_var(--color-hairline)]">
      {hints.map((hint) => (
        <span key={hint.keys} className="flex items-center gap-1.5">
          <kbd className="rounded-chip bg-fill px-1.5 py-0.5 font-sans text-label text-ink-2">
            {hint.keys}
          </kbd>
          {hint.label}
        </span>
      ))}
    </div>
  );
}
