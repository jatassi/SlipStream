import { sentenceOf } from "../api/errors";
import type { ImportsEntry, ImportsOutcome, ImportsOutcomeProblem } from "../api/gen/model";
import {
  ignorePendingImports,
  unignorePendingImports,
  useAddPendingImports,
} from "../api/gen/query/imports/imports";
import { toast } from "../ui/toast";
import { named } from "./words";

const counted = new Intl.NumberFormat("en");

type Failure = { readonly entry: ImportsEntry; readonly problem: ImportsOutcomeProblem };

function addedWords(added: readonly ImportsEntry[]): string {
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

function told(entries: readonly ImportsEntry[], outcomes: readonly ImportsOutcome[]) {
  const outcomeOf = (entry: ImportsEntry) => outcomes.find(({ id }) => id === entry.id);
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
  const adding = useAddPendingImports();
  const add = (entries: readonly ImportsEntry[]) => {
    adding.mutate(
      { data: { ids: entries.map(({ id }) => id) } },
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

export async function restoring(entry: ImportsEntry) {
  try {
    await unignorePendingImports({ ids: [entry.id] });
    toast(`${entry.folder} needs you again.`);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

export async function ignoring(entry: ImportsEntry) {
  try {
    await ignorePendingImports({ ids: [entry.id] });
    toast(`Ignored ${entry.folder}. A library scan lists it again if its files change.`, {
      action: { label: "Undo", onClick: () => void restoring(entry) },
    });
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}
