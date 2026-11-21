import type { ReactNode } from "react";

import type { WorksContentType } from "../api/gen/model";
import { PagePending } from "../shell/page";
import { LibraryPage } from "./library-page";
import { type ViewScope, rememberView } from "./remembered";
import { type LibrarySearch, decodeView, encodeView } from "./view-codec";

export { rememberedSearch } from "./remembered";
export { librarySearch } from "./view-codec";
export { librarySwitches } from "./type-switch";

/** The Library's skeleton in the layout and poster size its address names, before the types load. */
export function LibraryPending({ search }: { search: LibrarySearch }) {
  const { layout, size } = decodeView(search, []);
  return layout === "grid" ? (
    <PagePending label="Loading the Library" shape="grid" posters={size} />
  ) : (
    <PagePending label="Loading the Library" shape="table" lines={2} />
  );
}

/** A Library whose view lives in the address, kept for this Library's next visit. */
export function RoutedLibrary({
  scope,
  types,
  search,
  onSearch,
  empty,
}: {
  scope: ViewScope;
  types: readonly WorksContentType[];
  search: LibrarySearch;
  onSearch: (search: LibrarySearch) => void;
  empty: ReactNode;
}) {
  return (
    <LibraryPage
      key={scope}
      types={types}
      view={decodeView(search, types)}
      onView={(next) => {
        const encoded = encodeView(next, types);
        rememberView(scope, encoded);
        onSearch(encoded);
      }}
      empty={empty}
    />
  );
}
