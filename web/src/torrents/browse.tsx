import { type ReactNode, useDeferredValue, useState } from "react";

import type { DownloadclientsTorrentList } from "../api/gen/model";
import type { Clock } from "../format";
import { useShell } from "../ui/shell-context";
import { NoMatch } from "./empty";
import { type Row, type Tab, matches, tabCount } from "./model";
import type { Names } from "./names";
import { usePicking } from "./picking";
import { type RowSeams, TorrentTable } from "./table";
import { Toolbar } from "./toolbar";

export type TorrentsSearch = {
  readonly tab?: Tab | undefined;
  readonly node?: string | undefined;
  readonly type?: string | undefined;
  readonly q?: string | undefined;
};

const rowEstimate = { desktop: 60, phone: 220 } as const;

export type Seams = RowSeams & {
  readonly renderBulk?: ((picked: readonly Row[], clear: () => void) => ReactNode) | undefined;
};

type BrowseProps = Seams & {
  list: DownloadclientsTorrentList;
  search: TorrentsSearch;
  onSearchChange: (change: Partial<TorrentsSearch>) => void;
  names: Names;
  clock: Clock;
};

// The search box's text: deferred for the filter, and kept in the address.
function useDraft(
  initial: string | undefined,
  onSearchChange: (change: Partial<TorrentsSearch>) => void,
) {
  const [draft, setDraft] = useState(initial ?? "");
  const write = (next: string) => {
    setDraft(next);
    onSearchChange({ q: next === "" ? undefined : next });
  };
  return { draft, setDraft, q: useDeferredValue(draft), write };
}

export function Browse({
  list,
  search,
  onSearchChange,
  names,
  clock,
  renderBulk,
  ...seams
}: BrowseProps) {
  const shell = useShell();
  const { draft, setDraft, q, write } = useDraft(search.q, onSearchChange);
  const tab = search.tab ?? (shell === "phone" ? "all" : "downloading");
  const filter = { tab, node: search.node, type: search.type, q };
  const shown = list.rows.filter((row) => matches(row, filter, names.work(row.workId)));
  const { picking, picked, clear: unpick } = usePicking(shown);
  const narrowed = search.node !== undefined || search.type !== undefined || q !== "";
  const clear = () => {
    setDraft("");
    onSearchChange({ node: undefined, type: undefined, q: undefined });
  };
  return (
    <>
      <Toolbar
        tab={tab}
        counts={(each) => tabCount(list.counts, each)}
        rows={list.rows}
        node={search.node}
        type={search.type}
        q={draft}
        onQChange={write}
        onChange={onSearchChange}
      />
      {renderBulk !== undefined && picked.length > 0 && renderBulk(picked, unpick)}
      {shown.length === 0 ? (
        <NoMatch onClear={narrowed ? clear : undefined} />
      ) : (
        <TorrentTable
          rows={shown}
          estimate={rowEstimate[shell]}
          names={names}
          clock={clock}
          picking={renderBulk === undefined ? undefined : picking}
          {...seams}
        />
      )}
    </>
  );
}
