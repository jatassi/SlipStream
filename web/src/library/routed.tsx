import type { ReactNode } from "react";

import type { WorksContentType } from "../api/gen/model";
import { LibraryPage } from "./library-page";
import { type ViewScope, rememberView } from "./remembered";
import { type LibrarySearch, decodeView, encodeView } from "./view-codec";

export { rememberedSearch } from "./remembered";
export { librarySearch } from "./view-codec";

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
