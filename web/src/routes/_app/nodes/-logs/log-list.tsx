import type { ReactNode } from "react";

import { Empty } from "../../../../shell/empty";
import { Button } from "../../../../ui/button";
import { VirtualRows } from "../../../../ui/virtual-rows";
import { type LogRow, rowKey } from "./buffer";

const hasLines = (rows: readonly LogRow[]) => rows.some((row) => row.kind === "line");

/** The rows a filter keeps, or why there are none. */
export function LogList({
  held,
  rows,
  onClearFilters,
  ...list
}: {
  held: readonly LogRow[];
  rows: readonly LogRow[];
  onClearFilters: () => void;
  follow: boolean;
  onFollowChange: (follow: boolean) => void;
  renderRow: (row: LogRow) => ReactNode;
}) {
  if (!hasLines(held)) {
    return <Empty>No log lines yet</Empty>;
  }
  if (!hasLines(rows)) {
    return (
      <Empty action={<Button onClick={onClearFilters}>Clear filters</Button>}>
        No lines match your filters
      </Empty>
    );
  }
  return <VirtualRows label="Log" rows={rows} rowKey={rowKey} estimate={32} {...list} />;
}
