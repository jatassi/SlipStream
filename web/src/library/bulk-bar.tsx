import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { countWords } from "../format";
import { Button } from "../ui/button";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { useShell } from "../ui/shell-context";
import { type BulkAction, useBulkActions } from "./bulk-actions";
import { BulkDialog, type BulkDialogKind } from "./bulk-dialogs";
import type { Picking } from "./picking";

const anyWork = { one: "Work", many: "Works" } as const;

// It floats at the bottom of the view, above a phone's tabs, and comes after
// the Works so that opening it never moves them.
const floating =
  "sticky z-10 mt-4 flex items-center gap-2 rounded-card bg-popover py-2 pl-4 shadow-float ring-1 ring-hairline ring-inset backdrop-blur-[30px] backdrop-saturate-[1.7]";

const placed = {
  phone: "bottom-[calc(env(safe-area-inset-bottom)+57px)] pr-2",
  desktop: "bottom-3 flex-wrap pr-4",
} as const;

const menuLabel = "Actions for the picked Works";

function BulkButtons({ actions, oneType }: { actions: readonly BulkAction[]; oneType: boolean }) {
  return (
    <>
      {actions
        .filter(({ oneTypeOnly }) => oneType || !oneTypeOnly)
        .map(({ label, icon, busy, danger, onSelect }) => (
          <Button
            key={label}
            icon={icon}
            busy={busy}
            look={danger ? "ghost" : "default"}
            onClick={onSelect}
          >
            {label}
          </Button>
        ))}
      {!oneType && (
        <span className="text-meta text-ink-3">
          Pick Works of one Content Type to change their Profile or library root.
        </span>
      )}
    </>
  );
}

function BulkMenu({
  actions,
  oneType,
  onSelectAll,
}: {
  actions: readonly BulkAction[];
  oneType: boolean;
  onSelectAll: () => void;
}) {
  const items: MenuEntry[] = [
    { label: "Select all", onSelect: onSelectAll },
    ...(actions.length === 0 ? [] : ["separator" as const]),
    ...actions.map(({ label, icon, busy, danger, oneTypeOnly, onSelect }): MenuEntry => ({
      label,
      icon,
      tone: danger ? "danger" : "default",
      disabled: busy || (oneTypeOnly && !oneType),
      onSelect,
    })),
  ];
  return (
    <Menu
      label={menuLabel}
      trigger={<IconButton icon={MoreHorizontalIcon} label={menuLabel} />}
      items={items}
    />
  );
}

function typeOf(picked: Picking["picked"]) {
  const [first] = picked;
  return first !== undefined && picked.every((work) => work.type.id === first.type.id)
    ? first.type
    : undefined;
}

/**
 * What the picked Works can do together, each one request that answers per
 * Work. An action acts on every pick, the ones the filters hide included,
 * as the bar says. On a phone the actions sit in one menu, so the bar
 * stays one row.
 */
export function BulkBar({ picking }: { picking: Picking }) {
  const [dialog, setDialog] = useState<BulkDialogKind>();
  const phone = useShell() === "phone";
  const { picked, hidden } = picking;
  const actions = useBulkActions(picked, setDialog);
  const type = typeOf(picked);
  const oneType = type !== undefined;
  return (
    <div
      role="toolbar"
      aria-label="Picked Works"
      className={`${floating} ${placed[phone ? "phone" : "desktop"]}`}
    >
      <span className="mr-auto min-w-0 text-ink-1 tabular-nums">
        {picked.length === 0
          ? "Pick Works to change them together."
          : `${countWords(picked.length, type?.work ?? anyWork)} picked`}
        {hidden > 0 && <span className="text-ink-3">{` · ${hidden} hidden by filters`}</span>}
      </span>
      {phone ? (
        <BulkMenu
          actions={picked.length === 0 ? [] : actions}
          oneType={oneType}
          onSelectAll={picking.toggleAll}
        />
      ) : (
        <>
          <Button look="ghost" onClick={picking.toggleAll}>
            Select all
          </Button>
          {picked.length > 0 && <BulkButtons actions={actions} oneType={oneType} />}
        </>
      )}
      <Button look="ghost" onClick={picking.done}>
        Done
      </Button>
      <BulkDialog
        kind={dialog}
        type={type}
        picked={picked}
        onClose={() => {
          setDialog(undefined);
        }}
      />
    </div>
  );
}
