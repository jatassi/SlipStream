import { useQuery } from "@tanstack/react-query";

import type { IndexersDefinitionForm, IndexersIndexer } from "../api/gen/model";
import {
  getCatalogDefinition,
  getGetCatalogDefinitionQueryKey,
} from "../api/gen/query/indexers/indexers";
import { type PortForm, portFormOf } from "./port-shape";

/** The form an Indexer's settings are drawn from: its definition's, or its port's. */
export type SettingsForm =
  | { readonly kind: "catalog"; readonly form: IndexersDefinitionForm }
  | { readonly kind: "native"; readonly form: PortForm };

const formIdOf = (indexer: IndexersIndexer) => indexer.definition?.id ?? indexer.port?.id;

/**
 * The form of a catalog or native Indexer, read as Add Indexer reads it. A
 * feed has none, and a definition the catalog no longer lists has none
 * either, so its settings can't be drawn.
 */
export function useSettingsForm(indexer: IndexersIndexer) {
  const id = formIdOf(indexer) ?? "";
  return useQuery({
    queryKey: getGetCatalogDefinitionQueryKey(id),
    queryFn: ({ signal }) => getCatalogDefinition(id, { signal }),
    enabled: id !== "",
    select: (form): SettingsForm =>
      indexer.kind === "native"
        ? { kind: "native", form: portFormOf(form) }
        : { kind: "catalog", form },
  });
}
