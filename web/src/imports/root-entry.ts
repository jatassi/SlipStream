import type {
  ImportsImportEntry,
  ImportsImportFile,
  ImportsReason,
  ImportsRootLabel,
} from "../api/gen/model";
import { useListImportsSuspense } from "../api/gen/query/imports/imports";

/** A waiting file of a root Import, with the reason it waits for. */
export type RootFile = ImportsImportFile & { readonly reason: ImportsReason };

/** A root Import as Pending imports shows it: its root and its waiting files. */
export type RootEntry = Omit<ImportsImportEntry, "root" | "files"> & {
  readonly root: ImportsRootLabel;
  readonly files: RootFile[];
};

const isWaiting = (file: ImportsImportFile): file is RootFile =>
  file.outcome === "waiting" && file.reason !== null;

/** The root Imports of `imports`, each with only the files that wait. */
export const rootEntriesOf = (imports: readonly ImportsImportEntry[]): RootEntry[] =>
  imports.flatMap(({ root, files, ...entry }) =>
    entry.source === "root" && root !== null
      ? [{ ...entry, root, files: files.filter((file) => isWaiting(file)) }]
      : [],
  );

/** Pending imports' root leads and root Imports, of the needs-you or the ignored view. */
export function useRootImports(params: { source: "root"; ignored?: boolean }) {
  const { leads, imports } = useListImportsSuspense(params).data;
  return { leads, entries: rootEntriesOf(imports) };
}
