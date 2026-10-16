import { useState } from "react";

import { useShell } from "../../../../ui/shell-context";
import { rowKey } from "./buffer";
import { type LogFilter, noFilter, subsystemsOf, visibleRows } from "./filter";
import { LogBar } from "./log-bar";
import { LogList } from "./log-list";
import { LogRowView } from "./log-row";
import { LogStatus } from "./log-status";
import { useLogStream } from "./use-log-stream";

const heights = {
  desktop: "h-[calc(100dvh-15rem)]",
  phone: "h-[calc(100dvh-13rem)]",
} as const;

function toggled(keys: ReadonlySet<string>, key: string): ReadonlySet<string> {
  const next = new Set(keys);
  if (!next.delete(key)) {
    next.add(key);
  }
  return next;
}

/** One Node's log as it is written, following the tail until the Admin scrolls away. */
export function LogViewer({ nodeId, cap }: { nodeId: string; cap: number }) {
  const { view, dispatch } = useLogStream(nodeId, cap);
  const [filter, setFilter] = useState<LogFilter>(noFilter);
  const [follow, setFollow] = useState(true);
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
  return (
    <div className={`flex min-h-96 flex-col gap-3 ${heights[useShell()]}`}>
      <LogBar
        nodeId={nodeId}
        paused={view.paused}
        dispatch={dispatch}
        filter={filter}
        onFilterChange={setFilter}
        subsystems={subsystemsOf(view.lines)}
      />
      <LogList
        held={view.lines}
        rows={visibleRows(view.lines, filter)}
        onClearFilters={() => {
          setFilter(noFilter);
        }}
        follow={follow}
        onFollowChange={setFollow}
        renderRow={(row) => (
          <LogRowView
            row={row}
            expanded={expanded.has(rowKey(row))}
            onToggle={() => {
              setExpanded((keys) => toggled(keys, rowKey(row)));
            }}
          />
        )}
      />
      <LogStatus
        view={view}
        follow={follow}
        onJump={() => {
          setFollow(true);
        }}
      />
    </div>
  );
}
