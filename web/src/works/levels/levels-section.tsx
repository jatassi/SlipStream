import { useMemo, useState } from "react";

import type { WorksContentType, WorksWorkView } from "../../api/gen/model";
import { CellPopover, useCellPopover } from "../../ui/cell-popover";
import { Section } from "../../ui/section";
import { useShell } from "../../ui/shell-context";
import type { MatrixShown } from "../page/matrix-model";
import type { WorkReads } from "../page/reads";
import { CellDetail, type PoppedCell, cellLabel } from "./cell-detail";
import { ItemDetails } from "./item-details";
import { ItemSheet } from "./item-sheet";
// oxlint-disable-next-line import/max-dependencies -- the section composes its rows, cell popover, Item Details and phone sheet
import { type LevelView, levelContext, levelsOf, viewOf } from "./model";
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
      levelRows(levels, ({ level: { id }, opens }) => {
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
      const { id } = level.level;
      const next = new Map(prev).set(id, !(prev.get(id) ?? firstShown.get(id) ?? level.opens));
      openedByWork.set(work.id, next);
      return next;
    });
  };
  return { context, levels, rows, toggle };
}

type Picked = { readonly id: string; readonly as: "details" | "sheet" };

function useActions(workId: string, toggle: (level: LevelView) => void) {
  const monitor = useMonitorParts(workId);
  const [picked, setPicked] = useState<Picked>();
  const rowActions: Drawn["rowActions"] = {
    monitor: (view, on) => {
      monitor({ levelIds: [], itemIds: [view.item.id], on });
    },
    details: (view) => {
      setPicked({ id: view.item.id, as: "details" });
    },
    open: (view) => {
      setPicked({ id: view.item.id, as: "sheet" });
    },
  };
  const headerActions: Drawn["headerActions"] = {
    toggle,
    monitor: (level, on) => {
      monitor({ levelIds: [level.level.id], itemIds: [], on });
    },
  };
  // The sheet's Details closes the sheet after it opens Details, so each closes only itself.
  const close = (as: Picked["as"]) => () => {
    setPicked((prev) => (prev?.as === as ? undefined : prev));
  };
  return { picked, close, rowActions, headerActions };
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
  const { picked, close, rowActions, headerActions } = useActions(work.id, toggle);
  const found = picked && viewOf(levels, picked.id);
  return (
    <Section title={type.levels[0]?.many ?? type.item.many}>
      <VirtualLevels
        rows={rows}
        drawn={{ phone, context, controls: popover.controls, rowActions, headerActions }}
      />
      <CellPopover state={popover} label={cellLabel}>
        {(popped) => <CellDetail popped={popped} shown={shown} />}
      </CellPopover>
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
