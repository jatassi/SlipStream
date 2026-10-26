import { useState } from "react";

import { countWords } from "../format";
import { Button } from "../ui/button";
import { BulkActions } from "./bulk-actions";
import { BulkDialog, type BulkDialogKind } from "./bulk-dialogs";
import type { Picking } from "./picking";

const anyWork = { one: "Work", many: "Works" } as const;

/**
 * What the picked Works can do together, each one request that answers per
 * Work. An action acts on every pick, the ones the filters hide included,
 * as the bar says.
 */
export function BulkBar({ picking }: { picking: Picking }) {
  const [dialog, setDialog] = useState<BulkDialogKind>();
  const { picked, hidden } = picking;
  const [first] = picked;
  const type =
    first !== undefined && picked.every((work) => work.type.id === first.type.id)
      ? first.type
      : undefined;
  return (
    <div
      role="toolbar"
      aria-label="Picked Works"
      className="mb-4 flex flex-wrap items-center gap-2 rounded-card bg-glass px-4 py-2 ring-1 ring-hairline ring-inset"
    >
      <span className="mr-auto text-ink-1 tabular-nums">
        {picked.length === 0
          ? "Pick Works to change them together."
          : `${countWords(picked.length, type?.work ?? anyWork)} picked`}
        {hidden > 0 && <span className="text-ink-3">{` · ${hidden} hidden by filters`}</span>}
      </span>
      <Button look="ghost" onClick={picking.toggleAll}>
        Select all
      </Button>
      {picked.length > 0 && (
        <BulkActions picked={picked} oneType={type !== undefined} onDialog={setDialog} />
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
