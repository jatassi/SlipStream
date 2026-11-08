import { useState } from "react";

import { isApiError, sentenceOf } from "../api/errors";
import type { FormatsDraft, FormatsParsed } from "../api/gen/model";
import { parseReleases } from "../api/gen/query/formats/formats";
import type { Problems } from "./draft";
import { type TesterKind, draftProblems, namesIn, namesProblem } from "./tester-words";

export type WorkRef = { readonly id: string; readonly label: string };

export type TesterInput = {
  readonly text: string;
  readonly kind: TesterKind;
  /** The Work every name is judged against; without one, each name is matched by its title. */
  readonly work?: WorkRef | undefined;
};

export type Outcome =
  | { readonly state: "idle" }
  | { readonly state: "results"; readonly results: readonly FormatsParsed[] }
  | { readonly state: "refused"; readonly sentence: string };

export type Testing = {
  readonly input: TesterInput;
  readonly outcome: Outcome;
  readonly busy: boolean;
  readonly edit: (input: TesterInput) => void;
  readonly test: (input: TesterInput, draft?: FormatsDraft) => void;
};

export function useTesting(onDraftRefused?: (problems: Problems) => void): Testing {
  const [input, setInput] = useState<TesterInput>({ text: "", kind: "release" });
  const [outcome, setOutcome] = useState<Outcome>({ state: "idle" });
  const [busy, setBusy] = useState(false);
  const run = async (tested: TesterInput, draft: FormatsDraft | undefined) => {
    const names = namesIn(tested.text);
    if (names.length === 0 || namesProblem(names, tested.kind) !== undefined) {
      return;
    }
    setBusy(true);
    setOutcome((shown) => (shown.state === "refused" ? { state: "idle" } : shown));
    try {
      const { results } = await parseReleases({
        names,
        kind: tested.kind,
        ...(tested.work !== undefined && { workId: tested.work.id }),
        ...(draft !== undefined && { draft }),
      });
      setOutcome({ state: "results", results });
    } catch (error) {
      if (draft !== undefined && isApiError(error) && error.errors.length > 0) {
        onDraftRefused?.(draftProblems(error));
      }
      setOutcome({ state: "refused", sentence: sentenceOf(error) });
    } finally {
      setBusy(false);
    }
  };
  return {
    input,
    outcome,
    busy,
    edit: setInput,
    test: (tested, draft) => {
      setInput(tested);
      void run(tested, draft);
    },
  };
}
