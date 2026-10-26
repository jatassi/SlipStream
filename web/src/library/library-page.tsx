import type { ReactNode } from "react";

import type { WorksContentType } from "../api/gen/model";
import { BulkBar } from "./bulk-bar";
import { columnsFor } from "./columns";
import { useRefreshAll } from "./refresh-all";
import { Results } from "./results";
import { Toolbar } from "./toolbar";
import { useArranged } from "./use-arranged";
import { type LibraryView, defaultView, nounOf } from "./view";

type LibraryProps = {
  readonly types: readonly WorksContentType[];
  readonly view: LibraryView;
  readonly onView: (view: LibraryView) => void;
  /** What an empty Library offers, such as adding its first Work. */
  readonly empty: ReactNode;
};

/**
 * The Library of one Content Type, or of every one on "All": a poster grid
 * or a table over the whole projection, sorted, sectioned and filtered in
 * the browser, with bulk edit over the picked Works.
 */
export function LibraryPage({ types, view, onView, empty }: LibraryProps) {
  const { ctx, scope, arranged, works, picking } = useArranged(types, view);
  const onRefreshAll = useRefreshAll(types);
  const noun = nounOf(types);
  const columns = columnsFor(types);
  const change = (next: Partial<LibraryView>) => {
    onView({ ...view, ...next });
  };
  const clear = () => {
    change({ filters: defaultView(types).filters, q: "" });
  };
  if (arranged.total === 0) {
    return empty;
  }
  return (
    <>
      <Toolbar
        scope={scope}
        view={view}
        onView={change}
        works={works}
        ctx={ctx}
        columns={columns}
        noun={noun}
        onSelect={picking.select}
        onRefreshAll={onRefreshAll}
      />
      {picking.selecting && <BulkBar picking={picking} />}
      <Results
        arranged={arranged}
        view={view}
        columns={columns.filter(({ key }) => view.columns.includes(key))}
        names={ctx}
        picking={picking}
        noun={noun}
        onClear={clear}
      />
    </>
  );
}
