import { type ReactNode, Suspense } from "react";

import type { FilesRenameRow, FilesRowBlock, FilesWorkLine } from "../api/gen/model";
import { useProjection } from "../library/projection";
import { Checkbox } from "../ui/checkbox";
import { Chevron } from "../ui/chevron";
import { dataFont } from "../ui/data";
import { type Part, pathDiff } from "./path-diff";
import {
  type RowState,
  type RowsOf,
  type Ticks,
  isTicked,
  rowStates,
  tickFile,
  tickWork,
  tickedIn,
  workTick,
} from "./plan-model";
import { blockWords, companionWords, groupWords, heldWords, workWords } from "./renames-words";

const nameOf = (path: string) => path.slice(path.lastIndexOf("/") + 1);

function Path({ parts, look }: { parts: readonly Part[]; look: "from" | "to" }) {
  const Tag = look === "from" ? "del" : "ins";
  return (
    <Tag
      className={`${dataFont} text-data no-underline ${look === "from" ? "text-ink-3" : "text-ink-1"}`}
    >
      {parts.map(({ text, changed }, index) =>
        changed ? (
          <mark
            // oxlint-disable-next-line react/no-array-index-key -- a path's parts never move
            key={index}
            className={`rounded-[3px] text-inherit ${look === "from" ? "bg-fail/20 line-through decoration-ink-4" : "bg-live-fill"}`}
          >
            {text}
          </mark>
        ) : (
          // oxlint-disable-next-line react/no-array-index-key -- a path's parts never move
          <span key={index}>{text}</span>
        ),
      )}
    </Tag>
  );
}

const blockLook = "text-warn-ink";

/** The other Work of a clash by its title, as the Library holds it. */
function Clash({ block, workId, other }: { block: FilesRowBlock; workId: string; other: string }) {
  const title = useProjection().byId.get(other)?.label;
  return <p className={blockLook}>{blockWords(block, workId, title)}</p>;
}

function Block({ block, workId }: { block: FilesRowBlock; workId: string }) {
  const other = block.otherWorkId;
  if (other === undefined || other === workId) {
    return <p className={blockLook}>{blockWords(block, workId, undefined)}</p>;
  }
  return (
    <Suspense fallback={null}>
      <Clash block={block} workId={workId} other={other} />
    </Suspense>
  );
}

function FileRow({
  workId,
  row,
  state,
  ticked,
  onTick,
}: {
  workId: string;
  row: FilesRenameRow;
  state: RowState;
  ticked: boolean;
  onTick: (on: boolean) => void;
}) {
  const diff = pathDiff(row.from, row.to);
  const meta = "text-meta text-ink-3 in-data-[density=touch]:text-body-touch";
  return (
    <li className="flex items-start gap-3">
      <Checkbox
        label={nameOf(row.from)}
        hideLabel
        checked={state === "renames" && ticked}
        disabled={state !== "renames"}
        onCheckedChange={onTick}
      />
      <div className="flex min-w-0 flex-col gap-0.5 py-1 [overflow-wrap:anywhere]">
        <Path
          parts={row.to === "" ? [{ text: row.from, changed: false }] : diff.from}
          look="from"
        />
        {row.to !== "" && <Path parts={diff.to} look="to" />}
        {row.companions.length > 0 && <p className={meta}>{companionWords(row.companions)}</p>}
        {state === "renames" && row.group !== 0 && <p className={meta}>{groupWords}</p>}
        {state === "held" && <p className={meta}>{heldWords}</p>}
        {row.blocked !== undefined && <Block block={row.blocked} workId={workId} />}
      </div>
    </li>
  );
}

export function FileRows({
  line,
  rows,
  ticks,
  onTicks,
}: {
  line: FilesWorkLine;
  rows: readonly FilesRenameRow[];
  ticks: Ticks;
  onTicks: (ticks: Ticks) => void;
}) {
  const states = rowStates(rows);
  return (
    <ul aria-label={`Files of ${line.title}`} className="flex flex-col gap-2 pt-1 pb-2 pl-7">
      {rows.map((row) => (
        <FileRow
          key={row.fileId}
          workId={line.workId}
          row={row}
          state={states.get(row.fileId) ?? "renames"}
          ticked={isTicked(ticks, line.workId, row.fileId)}
          onTick={(on) => {
            onTicks(tickFile(ticks, line.workId, rows, row.fileId, on));
          }}
        />
      ))}
    </ul>
  );
}

function Opener({
  title,
  open,
  onOpenChange,
}: {
  title: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <button
      type="button"
      aria-expanded={open}
      aria-label={`Files of ${title}`}
      {...(open && { "data-panel-open": "" })}
      onClick={() => {
        onOpenChange(!open);
      }}
      className="group flex size-(--control-height) shrink-0 items-center justify-center rounded-item text-ink-2 hover:text-ink-1"
    >
      <Chevron />
    </button>
  );
}

/** One Work of a plan: its tick, what it renames, and its files once opened. */
export function WorkRow({
  line,
  rows,
  ticks,
  open,
  onOpenChange,
  onTicks,
  children,
}: {
  line: FilesWorkLine;
  rows: RowsOf;
  ticks: Ticks;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onTicks: (ticks: Ticks) => void;
  /** Its files' rows, or where they stand while read. */
  children: ReactNode;
}) {
  const tick = workTick(ticks, line, rows);
  return (
    <div className="flex flex-col border-b border-hairline">
      <div className="flex items-center gap-3">
        <div className="flex min-w-0 flex-1 flex-col pb-1.5">
          <Checkbox
            label={line.title}
            checked={tick === "on"}
            indeterminate={tick === "some"}
            disabled={line.rename === 0}
            onCheckedChange={(on) => {
              onTicks(tickWork(ticks, line.workId, on));
            }}
          />
          <span className="-mt-1.5 pl-6.5 text-meta text-ink-3 tabular-nums in-data-[density=touch]:pl-8.5 in-data-[density=touch]:text-body-touch">
            {workWords(tickedIn(ticks, line, rows), line.rename, line.blocked)}
          </span>
        </div>
        <Opener title={line.title} open={open} onOpenChange={onOpenChange} />
      </div>
      {open && children}
    </div>
  );
}
