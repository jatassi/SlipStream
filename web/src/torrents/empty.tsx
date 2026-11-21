import { Link } from "@tanstack/react-router";

import type { DownloadclientsCounts } from "../api/gen/model";
import { countWords } from "../format";
import { Button } from "../ui/button";
import { Empty } from "../ui/empty";
import { useShell } from "../ui/shell-context";
import { type Tab, tabCount, tabLabels, tabOrder } from "./model";

export function NoTorrents() {
  return (
    <Empty
      look="sheet"
      action={
        <Button nativeButton={false} render={<Link to="/library" />}>
          Open the Library
        </Button>
      }
    >
      No torrents yet. Open a Work and choose Search to grab a Release. Its torrent shows here until
      you remove it.
    </Empty>
  );
}

const torrents = { one: "torrent", many: "torrents" } as const;

const tabWords: {
  readonly [T in Tab]: { readonly nothing: string; readonly there: (count: number) => string };
} = {
  downloading: {
    nothing: "Nothing is downloading.",
    there: (n) => `${countWords(n, torrents)} ${n === 1 ? "is" : "are"} downloading.`,
  },
  seeding: {
    nothing: "Nothing is seeding.",
    there: (n) => `${countWords(n, torrents)} ${n === 1 ? "is" : "are"} seeding.`,
  },
  problems: {
    nothing: "No torrent has a problem.",
    there: (n) => `${countWords(n, torrents)} ${n === 1 ? "has a problem" : "have problems"}.`,
  },
  "goal-met": {
    nothing: "No torrent has met its seeding goal.",
    there: (n) => `${countWords(n, torrents)} met ${n === 1 ? "its" : "their"} seeding goal.`,
  },
  all: {
    nothing: "No torrents here right now.",
    there: (n) => `There ${n === 1 ? "is" : "are"} ${countWords(n, torrents)} in all.`,
  },
};

/**
 * A tab with no rows: with filters set, a way to clear them. Otherwise what
 * the tab is missing, then the first other tab that has torrents, with a
 * button that opens it.
 */
export function TabEmpty({
  tab,
  counts,
  onClear,
  onChange,
}: {
  tab: Tab;
  counts: DownloadclientsCounts;
  onClear: (() => void) | undefined;
  onChange: (change: { readonly tab: Tab }) => void;
}) {
  const shell = useShell();
  if (onClear !== undefined) {
    return (
      <Empty look="sheet" action={<Button onClick={onClear}>Clear filters</Button>}>
        No torrents match these filters.
      </Empty>
    );
  }
  const count = (each: Tab) => tabCount(counts, each);
  const other: Tab =
    tabOrder[shell].find((each) => each !== tab && each !== "all" && count(each) > 0) ?? "all";
  return (
    <Empty
      look="sheet"
      action={
        <Button
          onClick={() => {
            onChange({ tab: other });
          }}
        >
          {`Show ${tabLabels[other]}`}
        </Button>
      }
    >
      {`${tabWords[tab].nothing} ${tabWords[other].there(count(other))}`}
    </Empty>
  );
}
