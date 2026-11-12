import { sentenceOf } from "../api/errors";
import type { ImportsOutcome, ImportsOutcomeProblem, WorksLibraryRow } from "../api/gen/model";
import { type RootEntry, rootEntriesOf } from "./root-entry";
import {
  ignoreImports,
  unignoreImports,
  useActOnRootImport,
  useAddRootImports,
} from "../api/gen/query/imports/imports";
import { toast } from "../ui/toast";
import { filesWords, named } from "./words";

const counted = new Intl.NumberFormat("en");

type Failure = { readonly entry: RootEntry; readonly problem: ImportsOutcomeProblem };

function addedWords(added: readonly RootEntry[]): string {
  const [only] = added;
  return added.length === 1 && only?.suggestion
    ? `Added ${named(only.suggestion.label, only.suggestion.year)}.`
    : `Added ${counted.format(added.length)} Works.`;
}

function failedWords([first, ...rest]: readonly [Failure, ...Failure[]]): string {
  const what =
    rest.length === 0
      ? `${first.entry.folder} wasn't added.`
      : `${counted.format(rest.length + 1)} folders weren't added.`;
  const { title } = first.problem;
  return `${what} ${title.endsWith(".") ? title : `${title}.`}`;
}

function told(entries: readonly RootEntry[], outcomes: readonly ImportsOutcome[]) {
  const outcomeOf = (entry: RootEntry) => outcomes.find(({ id }) => id === entry.id);
  const added = entries.filter((entry) => outcomeOf(entry)?.outcome === "added");
  const failed = entries.flatMap((entry): Failure[] => {
    const outcome = outcomeOf(entry);
    return outcome?.outcome === "failed" ? [{ entry, problem: outcome.problem }] : [];
  });
  if (added.length > 0) {
    toast(addedWords(added));
  }
  const [first, ...rest] = failed;
  if (first !== undefined) {
    toast(failedWords([first, ...rest]), { kind: "fail" });
  }
}

/** Adds each entry's suggested Work, or attaches its files to that Work when it is in the library. */
export function useAdding() {
  const adding = useAddRootImports();
  const add = (entries: readonly RootEntry[]) => {
    adding.mutate(
      { data: { importIds: entries.map(({ id }) => id) } },
      {
        onSuccess: ({ outcomes }) => {
          told(entries, outcomes);
        },
        onError: (error) => {
          toast(sentenceOf(error), { kind: "fail" });
        },
      },
    );
  };
  return { add, busy: adding.isPending };
}

export async function restoring(entry: RootEntry) {
  try {
    await unignoreImports({ importIds: [entry.id] });
    toast(`${entry.folder} needs you again.`);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

export async function ignoring(entry: RootEntry) {
  try {
    await ignoreImports({ importIds: [entry.id] });
    toast(`Ignored ${entry.folder}. A library scan lists it again if its files change.`, {
      action: { label: "Undo", onClick: () => void restoring(entry) },
    });
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

function importedWords(count: number, work: string, left: number): string {
  const still = left === 0 ? "" : ` ${filesWords(left)} still ${left === 1 ? "waits" : "wait"}.`;
  return `Imported ${filesWords(count)} into ${work}.${still}`;
}

type ChosenFile = { readonly position: number; readonly itemIds: string[] };

/** Imports the chosen files of `entry` into `row`'s Work, then tells the Admin how many still wait. */
export function useImporting(entry: RootEntry, row: WorksLibraryRow, onDone: () => void) {
  const importing = useActOnRootImport();
  const mutate = (files: ChosenFile[]) => {
    importing.mutate(
      { id: entry.id, data: { action: "into", workId: row.id, files } },
      {
        onSuccess: (after) => {
          const left = rootEntriesOf([after]).at(0)?.files.length ?? 0;
          toast(importedWords(files.length, row.label, left));
          onDone();
        },
      },
    );
  };
  return {
    mutate,
    isPending: importing.isPending,
    error: importing.isError ? importing.error : null,
  };
}
