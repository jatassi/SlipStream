import { InformationCircleIcon, Search01Icon, SearchList01Icon } from "@hugeicons/core-free-icons";

import type { FilesCell } from "../../api/gen/model";
import { Sheet, SheetCancel, SheetGroup, SheetItem, SheetToggle } from "../../ui/sheet";
import { StateSquare, stateWords } from "../../ui/state-square";
import { CellFacts } from "../page/cell-facts";
import type { MatrixShown } from "../page/matrix-model";
import { cellLabel } from "./cell-detail";
import type { RowActions } from "./item-row";
import type { ItemView, LevelContext, LevelView } from "./model";

type Line = { readonly key: string; readonly label: string; readonly cell?: FilesCell };

function Cells({
  view,
  level,
  context,
  shown,
}: {
  view: ItemView;
  level: LevelView;
  context: LevelContext;
  shown: MatrixShown;
}) {
  const lines = level.columns.flatMap((column, index): Line[] => {
    const group = view.groups[index];
    if (group === undefined || group.kind === "unseen") {
      return column.seen ? [{ key: column.edition, label: `${column.name}, not seen yet` }] : [];
    }
    return context.targets.flatMap((target, at) => {
      const cell = group.cells[at];
      return cell === undefined
        ? []
        : [
            {
              key: `${column.edition} ${target.id}`,
              label: cellLabel({ code: view.code, column, target }),
              cell,
            },
          ];
    });
  });
  return (
    <ul aria-label="Cells" className="flex flex-col divide-y divide-hairline rounded-card bg-fill">
      {lines.map(({ key, label, cell }) => (
        <li key={key} className="flex items-start gap-3 px-4 py-3">
          {cell !== undefined && <StateSquare state={cell.state} quiet={view.quiet} />}
          <span className="flex min-w-0 flex-col gap-1">
            <span className="text-ink-1">
              {cell === undefined ? label : `${label}, ${stateWords[cell.state]}`}
            </span>
            {cell !== undefined && <CellFacts cell={cell} grabs={shown.grabs} />}
          </span>
        </li>
      ))}
    </ul>
  );
}

function Actions({ view, actions }: { view: ItemView; actions: RowActions }) {
  return (
    <SheetGroup>
      {!view.withdrawn && (
        <SheetToggle
          checked={view.item.monitored}
          onCheckedChange={(on) => {
            actions.monitor(view, on);
          }}
        >
          Monitor
        </SheetToggle>
      )}
      {!view.withdrawn && (
        <SheetItem
          icon={Search01Icon}
          onClick={() => {
            actions.search(view);
          }}
        >
          Search
        </SheetItem>
      )}
      {!view.withdrawn && (
        <SheetItem
          icon={SearchList01Icon}
          onClick={() => {
            actions.manual(view);
          }}
        >
          Manual search
        </SheetItem>
      )}
      <SheetItem
        icon={InformationCircleIcon}
        onClick={() => {
          actions.details(view);
        }}
      >
        Details
      </SheetItem>
    </SheetGroup>
  );
}

/** Phone: an Item's cells, its Monitor switch, Search, Manual search and Details, in a bottom sheet. */
export function ItemSheet({
  open,
  context,
  shown,
  actions,
  onClose,
}: {
  open: { view: ItemView; level: LevelView } | undefined;
  context: LevelContext;
  shown: MatrixShown;
  actions: RowActions;
  onClose: () => void;
}) {
  const title = open === undefined ? "" : `${open.view.code} ${open.view.item.label}`;
  return (
    <Sheet
      label={title}
      open={open !== undefined}
      onOpenChange={(next) => {
        if (!next) {
          onClose();
        }
      }}
    >
      {open !== undefined && (
        <>
          <p className="px-1 pb-1 text-title">{title}</p>
          <Cells view={open.view} level={open.level} context={context} shown={shown} />
          <Actions view={open.view} actions={actions} />
          <SheetCancel />
        </>
      )}
    </Sheet>
  );
}
