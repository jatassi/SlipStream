import { useState } from "react";

import { isApiError, sentenceOf } from "../api/errors";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import type { EditorSection } from "./editor";

export type Save<Draft> = {
  readonly label: string;
  /** Saves the draft; a refusal it throws shows under the button. */
  readonly onSave: (draft: Draft) => Promise<unknown>;
};

// A save can refuse with a sentence of its own, as a ConfirmDialog's action can.
const refusalOf = (error: unknown): string =>
  error instanceof Error && !isApiError(error) ? error.message : sentenceOf(error);

function SaveBar<Draft>({
  draft,
  save,
  problemsOf,
}: {
  draft: Draft;
  save: Save<Draft>;
  problemsOf: (draft: Draft) => string[];
}) {
  const [busy, setBusy] = useState(false);
  const [problems, setProblems] = useState<readonly string[]>([]);
  const submit = async () => {
    const found = problemsOf(draft);
    setProblems(found);
    if (found.length > 0) {
      return;
    }
    setBusy(true);
    try {
      await save.onSave(draft);
    } catch (error) {
      setProblems([refusalOf(error)]);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="mt-8 flex flex-col items-start gap-3">
      {problems.map((problem) => (
        <AlertLine key={problem} text={problem} />
      ))}
      <Button look="primary" busy={busy} onClick={() => void submit()}>
        {save.label}
      </Button>
    </div>
  );
}

/**
 * An editor built from a section table: one section per row over one draft.
 * Without `save` it edits but offers no Save.
 */
export function SectionEditor<Draft>({
  sections,
  initial,
  problemsOf,
  save,
}: {
  sections: readonly EditorSection<Draft>[];
  initial: Draft;
  problemsOf: (draft: Draft) => string[];
  save?: Save<Draft> | undefined;
}) {
  const [draft, setDraft] = useState(initial);
  return (
    <div className="flex max-w-[880px] flex-col">
      {sections.map((section) => (
        <Section key={section.id} id={section.id} title={section.label}>
          <section.component draft={draft} onChange={setDraft} />
        </Section>
      ))}
      {save !== undefined && <SaveBar draft={draft} save={save} problemsOf={problemsOf} />}
    </div>
  );
}
