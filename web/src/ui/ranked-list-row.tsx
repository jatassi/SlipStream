import {
  ArrowDown01Icon,
  ArrowUp01Icon,
  DragDropVerticalIcon,
  Link01Icon,
  MoreHorizontalIcon,
} from "@hugeicons/core-free-icons";
import { type DragControls, Reorder, useDragControls } from "motion/react";
import type { KeyboardEvent, ReactNode } from "react";

import { IconButton } from "./icon-button";
import { Menu, type MenuEntry } from "./menu";
import { type RankedRow, type Rows, type Step, canStep, canTie } from "./ranked-rows";
import { useDensity } from "./shell-context";

export type Control = "handle" | "up" | "down" | "actions";

/** What every row of one Ranked list reads and calls. */
export type ListState<T> = {
  readonly rows: Rows<T>;
  readonly ranks: ReadonlyMap<string, number>;
  readonly dragging: string | undefined;
  readonly hintId: string;
  readonly ties: boolean;
  readonly nameOf: (row: RankedRow<T>) => string;
  readonly renderRow: (value: T, rank: number) => ReactNode;
  readonly renderTrailing: ((value: T) => ReactNode) | undefined;
  readonly step: (id: string, by: Step, control: Control) => void;
  readonly tie: (id: string, tied: boolean) => void;
  readonly dragStart: (id: string) => void;
  readonly dragEnd: () => void;
};

type RowProps<T> = { readonly row: RankedRow<T>; readonly list: ListState<T> };

const stepKeys: Readonly<Record<string, Step>> = { ArrowUp: -1, ArrowDown: 1 };

const layout = { duration: 0.28, ease: [0.32, 0.72, 0, 1] } as const;

const rowLook =
  "relative flex items-center gap-3 px-4 py-2 first:rounded-t-card last:rounded-b-card not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline in-data-[density=touch]:not-first:before:left-4";

const draggedLook = "z-10 rounded-card bg-popover shadow-lg backdrop-blur-xl";

export function RankedListRow<T>({ row, list }: RowProps<T>) {
  const controls = useDragControls();
  const touch = useDensity() === "touch";
  return (
    <Reorder.Item
      as="li"
      value={row.id}
      data-row={row.id}
      dragListener={false}
      dragControls={controls}
      onDragStart={() => {
        list.dragStart(row.id);
      }}
      onDragEnd={list.dragEnd}
      transition={layout}
      className={`${rowLook} min-h-(--row-height) ${list.dragging === row.id ? draggedLook : ""}`}
    >
      <Lead row={row} list={list} controls={controls} />
      {row.kind === "item" ? (
        <div className="min-w-0 flex-1">
          {list.renderRow(row.value, list.ranks.get(row.id) ?? 0)}
        </div>
      ) : (
        <div className="flex min-w-0 flex-1 items-center gap-3">
          <span className="text-label-caps text-glow uppercase">{row.label}</span>
          <span aria-hidden="true" className="h-px flex-1 bg-glow" />
        </div>
      )}
      {row.kind === "item" && list.renderTrailing !== undefined && (
        <div className="shrink-0">{list.renderTrailing(row.value)}</div>
      )}
      {touch ? <RowMenu row={row} list={list} /> : <Actions row={row} list={list} />}
    </Reorder.Item>
  );
}

function Lead<T>({ row, list, controls }: RowProps<T> & { readonly controls: DragControls }) {
  const rank = list.ranks.get(row.id);
  return (
    <>
      {row.kind === "item" ? (
        <span
          className={`w-10 shrink-0 text-center text-numeral tabular-nums ${rank === 1 ? "text-ink-1" : "text-ink-3"}`}
        >
          <span className="sr-only">Rank </span>
          {rank}
        </span>
      ) : (
        <span aria-hidden="true" className="w-10 shrink-0" />
      )}
      <IconButton
        icon={DragDropVerticalIcon}
        label={`Move ${list.nameOf(row)}`}
        aria-describedby={list.hintId}
        data-control="handle"
        className="cursor-grab touch-none active:cursor-grabbing"
        onPointerDown={(event) => {
          controls.start(event);
        }}
        onKeyDown={(event: KeyboardEvent<HTMLElement>) => {
          const by = stepKeys[event.key];
          if (by !== undefined && canStep(list.rows, row.id, by)) {
            event.preventDefault();
            list.step(row.id, by, "handle");
          }
        }}
      />
    </>
  );
}

function Actions<T>({ row, list }: RowProps<T>) {
  const name = list.nameOf(row);
  const still = list.dragging === undefined;
  const move = (by: Step, control: Control) => () => {
    list.step(row.id, by, control);
  };
  return (
    <>
      {list.ties && row.kind === "item" && (
        <IconButton
          icon={Link01Icon}
          label={`Tie ${name} with the one above`}
          aria-pressed={row.tiedAbove}
          disabled={!still || !canTie(list.rows, row.id)}
          className="aria-pressed:text-glow"
          onClick={() => {
            list.tie(row.id, !row.tiedAbove);
          }}
        />
      )}
      {list.ties && row.kind === "marker" && (
        <span aria-hidden="true" className="size-(--icon-button-size) shrink-0" />
      )}
      <IconButton
        icon={ArrowUp01Icon}
        label={`Move ${name} up`}
        data-control="up"
        disabled={!still || !canStep(list.rows, row.id, -1)}
        onClick={move(-1, "up")}
      />
      <IconButton
        icon={ArrowDown01Icon}
        label={`Move ${name} down`}
        data-control="down"
        disabled={!still || !canStep(list.rows, row.id, 1)}
        onClick={move(1, "down")}
      />
    </>
  );
}

// At the Touch density the tie and the moves share one 44px button, so a
// row keeps its name on one line on a phone.
function RowMenu<T>({ row, list }: RowProps<T>) {
  const name = list.nameOf(row);
  const moves: MenuEntry[] = (
    [
      [-1, "Move up", ArrowUp01Icon],
      [1, "Move down", ArrowDown01Icon],
    ] as const
  ).flatMap(([by, label, icon]) =>
    canStep(list.rows, row.id, by)
      ? [
          {
            label,
            icon,
            onSelect: () => {
              list.step(row.id, by, "actions");
            },
          },
        ]
      : [],
  );
  const tie: MenuEntry[] =
    list.ties && row.kind === "item" && canTie(list.rows, row.id)
      ? [
          "separator",
          {
            label: row.tiedAbove ? "Untie from the one above" : "Tie with the one above",
            icon: Link01Icon,
            onSelect: () => {
              list.tie(row.id, !row.tiedAbove);
            },
          },
        ]
      : [];
  const items = [...moves, ...tie];
  return (
    <Menu
      label={`Actions for ${name}`}
      items={items}
      trigger={
        <IconButton
          icon={MoreHorizontalIcon}
          label={`Actions for ${name}`}
          data-control="actions"
          disabled={list.dragging !== undefined || items.length === 0}
        />
      }
    />
  );
}
