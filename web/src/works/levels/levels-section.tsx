import { useMemo, useState } from "react";

import type { WorksContentType, WorksWorkView } from "../../api/gen/model";
import { useStartSearch } from "../../grabs/search";
import { openManualSearch } from "../../manual/sheet-state";
import { CellPopover, type CellPopoverState, useCellPopover } from "../../ui/cell-popover";
import { Section } from "../../ui/section";
import { useShell } from "../../ui/shell-context";
import { type MatrixShown, cellScope } from "../page/matrix-model";
import type { WorkReads } from "../page/reads";
import { CellDetail, type PoppedCell, cellLabel } from "./cell-detail";
// oxlint-disable-next-line import/max-dependencies -- the section composes its rows, cell popover, Item Details and phone sheet
import { ItemDetails } from "./item-details";
import { ItemSheet } from "./item-sheet";
import { type ItemView, type LevelView, levelContext, levelsOf } from "./model";
import { levelRows } from "./rows";
import { useMonitorParts } from "./use-monitored";
import { type Drawn, VirtualLevels } from "./virtual-levels";

// Which levels an Admin opened or closed, per Work, for the rest of the session.
const openedByWork = new Map<string, ReadonlyMap<string, boolean>>();

function useLevels(type: WorksContentType, work: WorksWorkView, reads: WorkReads) {
  const { profile, targets, wants, cuts, cells } = reads;
  const context = useMemo(
    () => levelContext({ type, profile, targets, wants, cuts }),
    [type, profile, targets, wants, cuts],
  );
  const levels = useMemo(() => levelsOf(context, work, cells), [context, work, cells]);
  const [opened, setOpened] = useState(
    () => openedByWork.get(work.id) ?? new Map<string, boolean>(),
  );
  // A level keeps the open state it first showed with until the Admin
  // toggles it, so unmonitoring a level doesn't fold it away.
  const [firstShown] = useState(() => new Map<string, boolean>());
  const rows = useMemo(
    () =>
      levelRows(levels, ({ id, opens }) => {
        const shown = opened.get(id) ?? firstShown.get(id);
        if (shown !== undefined) {
          return shown;
        }
        firstShown.set(id, opens);
        return opens;
      }),
    [levels, opened, firstShown],
  );
  const toggle = (level: LevelView) => {
    setOpened((prev) => {
      const { id } = level;
      const next = new Map(prev).set(id, !(prev.get(id) ?? firstShown.get(id) ?? level.opens));
      openedByWork.set(work.id, next);
      return next;
    });
  };
  return { context, levels, rows, toggle };
}

/** An Item's view and its level's, by the Item's id. */
function viewOf(
  levels: readonly LevelView[],
  itemId: string,
): { view: ItemView; level: LevelView } | undefined {
  for (const level of levels) {
    const views = [
      ...level.entries.flatMap((entry) =>
        entry.kind === "item" ? [entry.item] : entry.fold.items,
      ),
      ...level.gone,
    ];
    const view = views.find(({ item }) => item.id === itemId);
    if (view !== undefined) {
      return { view, level };
    }
  }
  return undefined;
}

type Picked = { readonly id: string; readonly as: "details" | "sheet" };

function useActions(type: WorksContentType, workId: string, toggle: (level: LevelView) => void) {
  const monitor = useMonitorParts(workId);
  const search = useStartSearch(type.item);
  const [picked, setPicked] = useState<Picked>();
  const rowActions: Drawn["rowActions"] = {
    monitor: (view, on) => {
      monitor({ levelIds: [], itemIds: [view.item.id], on });
    },
    search: (view) => {
      search.start(workId, { levelIds: [], itemIds: [view.item.id] });
    },
    details: (view) => {
      setPicked({ id: view.item.id, as: "details" });
    },
    manual: (view) => {
      openManualSearch({ workId, scope: { itemIds: [view.item.id] } });
    },
    open: (view) => {
      setPicked({ id: view.item.id, as: "sheet" });
    },
  };
  const headerActions: Drawn["headerActions"] = {
    toggle,
    monitor: (level, on) => {
      monitor({ levelIds: [level.id], itemIds: [], on });
    },
    search: (level) => {
      search.start(workId, { levelIds: [level.id], itemIds: [] });
    },
    manual: (level) => {
      openManualSearch({ workId, scope: { levelIds: [level.id] } });
    },
  };
  // The sheet's Details closes the sheet after it opens Details, so each closes only itself.
  const close = (as: Picked["as"]) => () => {
    setPicked((prev) => (prev?.as === as ? undefined : prev));
  };
  return { picked, close, rowActions, headerActions };
}

/** The squares' one popover, whose Manual search closes it as the sheet opens. */
function SquarePopover({
  popover,
  shown,
}: {
  popover: CellPopoverState<PoppedCell>;
  shown: MatrixShown;
}) {
  return (
    <CellPopover state={popover} label={cellLabel}>
      {(popped) => (
        <CellDetail
          popped={popped}
          shown={shown}
          onManualSearch={() => {
            popover.controls.close();
            openManualSearch({ workId: shown.work.id, scope: cellScope(popped.cell) });
          }}
        />
      )}
    </CellPopover>
  );
}

/**
 * A many-Item Work's levels between Needs you and Files: one card per
 * level, open while anything in it is wanted and not met, its Items as
 * rows under one window virtualizer. Squares share one cell popover.
 */
export function LevelsSection({
  type,
  work,
  reads,
  shown,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  reads: WorkReads;
  shown: MatrixShown;
}) {
  const phone = useShell() === "phone";
  const { context, levels, rows, toggle } = useLevels(type, work, reads);
  const popover = useCellPopover<PoppedCell>();
  const { picked, close, rowActions, headerActions } = useActions(type, work.id, toggle);
  const found = picked && viewOf(levels, picked.id);
  return (
    <Section title={type.levels[0]?.many ?? type.item.many}>
      <VirtualLevels
        rows={rows}
        drawn={{ phone, context, controls: popover.controls, rowActions, headerActions }}
      />
      <SquarePopover popover={popover} shown={shown} />
      <ItemDetails
        type={type}
        view={picked?.as === "details" ? found?.view : undefined}
        onClose={close("details")}
      />
      {phone && (
        <ItemSheet
          open={picked?.as === "sheet" ? found : undefined}
          context={context}
          shown={shown}
          actions={rowActions}
          onClose={close("sheet")}
        />
      )}
    </Section>
  );
}
