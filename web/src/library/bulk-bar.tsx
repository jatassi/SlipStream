import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { countWords } from "../format";
import { Button } from "../ui/button";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { useShell } from "../ui/shell-context";
import { type BulkAction, useBulkActions, useBulkSearch } from "./bulk-actions";
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

const menuLabel = "Actions for the selected Works";

function BulkMenu({
  name,
  lead,
  actions,
  oneType,
}: {
  name: string;
  lead: readonly MenuEntry[];
  actions: readonly BulkAction[];
  oneType: boolean;
}) {
  const items: MenuEntry[] = [
    ...lead,
    ...(lead.length > 0 && actions.length > 0 ? ["separator" as const] : []),
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
      label={name}
      trigger={<IconButton icon={MoreHorizontalIcon} label={name} />}
      items={items}
    />
  );
}

function BulkButtons({ actions, oneType }: { actions: readonly BulkAction[]; oneType: boolean }) {
  return (
    <>
      {actions
        .filter(({ secondary, oneTypeOnly }) => !secondary && (oneType || !oneTypeOnly))
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
      <BulkMenu
        name="More actions"
        lead={[]}
        actions={actions.filter(({ secondary }) => secondary)}
        oneType={oneType}
      />
      {!oneType && (
        <span className="text-meta text-ink-3">
          Select Works of one kind of media to change their Profile or library root.
        </span>
      )}
    </>
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
 * as the bar says. On a phone the actions sit in one menu, and on a
 * desktop the secondary ones do, so the bar stays one row.
 */
export function BulkBar({ picking }: { picking: Picking }) {
  const [dialog, setDialog] = useState<BulkDialogKind>();
  const phone = useShell() === "phone";
  const { picked, hidden } = picking;
  const search = useBulkSearch(picked);
  const actions = useBulkActions(picked, setDialog, search.action);
  const type = typeOf(picked);
  const oneType = type !== undefined;
  return (
    <div
      role="toolbar"
      aria-label="Selected Works"
      className={`${floating} ${placed[phone ? "phone" : "desktop"]}`}
    >
      <span className="mr-auto min-w-0 text-ink-1 tabular-nums">
        {picked.length === 0
          ? "Select Works to change them together."
          : `${countWords(picked.length, type ? { one: type.work.one.toLowerCase(), many: type.work.many.toLowerCase() } : anyWork)} selected`}
        {hidden > 0 && <span className="text-ink-3">{` · ${hidden} hidden by filters`}</span>}
      </span>
      {phone ? (
        <BulkMenu
          name={menuLabel}
          lead={[{ label: "Select all", onSelect: picking.toggleAll }]}
          actions={picked.length === 0 ? [] : actions}
          oneType={oneType}
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
      {search.dialog}
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
