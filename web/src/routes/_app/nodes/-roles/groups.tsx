import { Alert02Icon, CancelCircleIcon } from "@hugeicons/core-free-icons";
import { useId } from "react";

import type {
  NodesBoardNode,
  NodesNode,
  NodesRoleBoard,
  NodesRoleEntry,
  NodesRoleGroup,
} from "../../../../api/gen/model";
import type { Clock } from "../../../../format";
import {
  type Draft,
  type Hint,
  hintOf,
  lockedOf,
  roleLine,
  withGroup,
  withRole,
} from "../../../../nodes/roles";
import { Checkbox } from "../../../../ui/checkbox";
import { Icon } from "../../../../ui/icon";
import { SwitchControl } from "../../../../ui/switch";

const hintLooks = {
  block: { icon: CancelCircleIcon, className: "text-fail-ink" },
  warn: { icon: Alert02Icon, className: "text-warn-ink" },
} as const;

function HintLine({ hint }: { hint: Hint }) {
  const { icon, className } = hintLooks[hint.tone];
  return (
    <p className={`flex items-start gap-1.5 text-meta ${className}`}>
      <Icon icon={icon} className="mt-0.5 size-3.5 shrink-0" />
      {hint.text}
    </p>
  );
}

type Row = {
  readonly entry: NodesRoleEntry;
  readonly ticked: boolean;
  readonly locked: boolean;
  readonly line: string | undefined;
  readonly hint: Hint | undefined;
};

function RoleRow({ row, onTick }: { row: Row; onTick: (on: boolean) => void }) {
  return (
    <li className="flex flex-col gap-0.5 px-4 py-2">
      <div className="flex flex-wrap items-center justify-between gap-x-3">
        <Checkbox
          label={row.entry.label}
          checked={row.ticked}
          disabled={row.locked}
          onCheckedChange={onTick}
        />
        {row.line !== undefined && <span className="text-meta text-ink-3">{row.line}</span>}
      </div>
      {row.locked && (
        <p className="text-meta text-ink-3">The Controller always holds {row.entry.label}.</p>
      )}
      {row.hint !== undefined && <HintLine hint={row.hint} />}
    </li>
  );
}

function GroupCard({
  group,
  rows,
  onGroup,
  onRole,
}: {
  group: NodesRoleGroup;
  rows: readonly Row[];
  onGroup: (on: boolean) => void;
  onRole: (row: Row, on: boolean) => void;
}) {
  const titleId = useId();
  const ticked = rows.filter((row) => row.ticked).length;
  return (
    <section
      aria-labelledby={titleId}
      className="overflow-hidden rounded-card bg-fill ring-1 ring-hairline ring-inset"
    >
      <div className="flex min-h-11 items-center gap-3 px-4">
        <h3 id={titleId} className="flex-1 font-semibold text-ink-1">
          {group.label}
        </h3>
        <span className="text-meta text-ink-3 tabular-nums">
          {ticked} of {rows.length}
        </span>
        <SwitchControl
          label={group.label}
          checked={ticked === rows.length}
          onCheckedChange={onGroup}
        />
      </div>
      <ul className="divide-y divide-hairline border-t border-hairline">
        {rows.map((row) => (
          <RoleRow
            key={row.entry.role}
            row={row}
            onTick={(on) => {
              onRole(row, on);
            }}
          />
        ))}
      </ul>
    </section>
  );
}

export type GroupsProps = {
  readonly board: NodesRoleBoard;
  readonly seat: NodesBoardNode;
  readonly node: NodesNode;
  readonly draft: Draft;
  readonly clock: Clock;
  readonly onDraft: (draft: Draft) => void;
};

/**
 * A switch per Role Group that ticks its Roles together, each Role still
 * tickable alone, with the state line of each Role the Node stands in and
 * why a Role may not do well there.
 */
export function RoleGroups({ board, seat, node, draft, clock, onDraft }: GroupsProps) {
  const locked = lockedOf(board, seat);
  const standing = { standing: node.standing, since: node.link.since };
  const rowsOf = (group: NodesRoleGroup): Row[] =>
    group.roles.map((entry) => {
      const view = seat.roles.find(({ role }) => role === entry.role);
      return {
        entry,
        ticked: draft.has(entry.role),
        locked: locked.has(entry.role),
        line: view === undefined ? undefined : roleLine(board, view, standing, clock),
        hint: hintOf(board, seat, entry),
      };
    });
  return board.groups.map((group) => (
    <GroupCard
      key={group.group}
      group={group}
      rows={rowsOf(group)}
      onGroup={(on) => {
        onDraft(withGroup(draft, group, on, locked));
      }}
      onRole={(row, on) => {
        onDraft(withRole(draft, row.entry.role, on));
      }}
    />
  ));
}
