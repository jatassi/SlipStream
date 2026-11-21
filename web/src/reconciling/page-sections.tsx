import { Link } from "@tanstack/react-router";

import type { ReconcilingPage, ReconcilingRootRow } from "../api/gen/model";
import { Button } from "../ui/button";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { ContinueWithoutRoot } from "./continue";
import { groupWords, rootLine, rootWaits, torrentsWords } from "./words";

function RootRow({ row, restoring }: { row: ReconcilingRootRow; restoring: boolean }) {
  return (
    <ListRow
      title={row.label}
      wrap
      detail={
        <div className="flex flex-col gap-2 pb-1 text-ink-2">
          <p>{rootLine(row)}</p>
          {restoring && rootWaits(row) && (
            <div className="flex flex-wrap gap-2 pt-1">
              {!row.saved && (
                <Button
                  size="small"
                  nativeButton={false}
                  render={<Link to="/reconciling/folders" />}
                >
                  Find its folder
                </Button>
              )}
              <ContinueWithoutRoot root={row} />
            </div>
          )}
        </div>
      }
    />
  );
}

export function RootsSection({ page }: { page: ReconcilingPage }) {
  if (page.roots.length === 0) {
    return null;
  }
  return (
    <Section title="Library roots">
      <List label="Library roots">
        {page.roots.map((row) => (
          <RootRow key={row.id} row={row} restoring={page.restore !== null} />
        ))}
      </List>
    </Section>
  );
}

export function ReviewSection({ page }: { page: ReconcilingPage }) {
  const total = page.review.reduce((sum, group) => sum + group.count, 0);
  if (total === 0) {
    return null;
  }
  return (
    <Section title="Review list" count={total}>
      <div className="flex flex-col items-start gap-3 text-ink-2">
        <ul className="flex max-w-[64ch] flex-col gap-1">
          {page.review.map((group) => (
            <li key={`${group.reason}:${group.found}`}>{groupWords(group).sentence}</li>
          ))}
        </ul>
        <Button nativeButton={false} render={<Link to="/reconciling/torrents" />}>
          {`Review ${torrentsWords(total)}`}
        </Button>
      </div>
    </Section>
  );
}
