import type { ReactNode } from "react";

import { Empty } from "../shell/empty";
import { Button } from "../ui/button";
import type { Arranged } from "./arrange";
import type { ColumnDef, Names } from "./columns";
import { LibraryGrid } from "./grid";
import type { Picking } from "./picking";
import { LibraryTable } from "./table";
import type { LibraryView } from "./view";

function ClearFilters({ onClear }: { onClear: () => void }) {
  return (
    <Button look="ghost" size="small" onClick={onClear}>
      Clear filters
    </Button>
  );
}

function Count({
  arranged,
  noun,
  children,
}: {
  arranged: Arranged;
  noun: string;
  children: ReactNode;
}) {
  return (
    <p className="mb-4 flex items-center gap-3 text-meta text-ink-3 tabular-nums">
      <span>{`${arranged.works.length} of ${arranged.total} ${noun}`}</span>
      {children}
    </p>
  );
}

/** The arranged Works as the view lays them out, with a count and Clear filters while narrowed. */
export function Results({
  arranged,
  view,
  columns,
  names,
  picking,
  noun,
  onClear,
}: {
  arranged: Arranged;
  view: LibraryView;
  columns: readonly ColumnDef[];
  names: Names;
  picking: Picking;
  noun: string;
  onClear: () => void;
}) {
  const filtered = Object.keys(view.filters).length > 0;
  const clear = <ClearFilters onClear={onClear} />;
  if (arranged.works.length === 0) {
    return (
      <Empty action={clear}>
        {filtered ? `No ${noun} match these filters.` : `No ${noun} match "${view.q}".`}
      </Empty>
    );
  }
  return (
    <>
      {(filtered || view.q !== "") && (
        <Count arranged={arranged} noun={noun}>
          {clear}
        </Count>
      )}
      {view.layout === "grid" ? (
        <LibraryGrid arranged={arranged} size={view.size} picking={picking} />
      ) : (
        <LibraryTable
          arranged={arranged}
          columns={columns}
          names={names}
          picking={picking}
          label={noun}
        />
      )}
    </>
  );
}
