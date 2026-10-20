import { useDeferredValue, useState } from "react";

import type { DownloadclientsTorrentList } from "../api/gen/model";
import type { Clock } from "../format";
import { useShell } from "../ui/shell-context";
import { NoMatch } from "./empty";
import { type Tab, matches, tabCount } from "./model";
import type { Names } from "./names";
import { type RowSeams, TorrentTable } from "./table";
import { Toolbar } from "./toolbar";

export type TorrentsSearch = {
  readonly tab?: Tab | undefined;
  readonly node?: string | undefined;
  readonly type?: string | undefined;
  readonly q?: string | undefined;
};

const rowEstimate = { desktop: 60, phone: 220 } as const;

type BrowseProps = RowSeams & {
  list: DownloadclientsTorrentList;
  search: TorrentsSearch;
  onSearchChange: (change: Partial<TorrentsSearch>) => void;
  names: Names;
  clock: Clock;
};

export function Browse({ list, search, onSearchChange, names, clock, ...seams }: BrowseProps) {
  const shell = useShell();
  const [draft, setDraft] = useState(search.q ?? "");
  const q = useDeferredValue(draft);
  const tab = search.tab ?? (shell === "phone" ? "all" : "downloading");
  const filter = { tab, node: search.node, type: search.type, q };
  const shown = list.rows.filter((row) => matches(row, filter, names.work(row.workId)));
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
        onQChange={(next) => {
          setDraft(next);
          onSearchChange({ q: next === "" ? undefined : next });
        }}
        onChange={onSearchChange}
      />
      {shown.length === 0 ? (
        <NoMatch onClear={narrowed ? clear : undefined} />
      ) : (
        <TorrentTable
          rows={shown}
          estimate={rowEstimate[shell]}
          names={names}
          clock={clock}
          {...seams}
        />
      )}
    </>
  );
}
