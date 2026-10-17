import { useState } from "react";

import { sentenceOf } from "../api/errors";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import { type TargetDraft, problemsOf } from "./draft";
import { targetSections } from "./editor";

export type SaveTarget = {
  readonly label: string;
  /** Saves the draft; a refusal it throws shows under the button. */
  readonly onSave: (draft: TargetDraft) => Promise<unknown>;
};

function useSave(draft: TargetDraft, save: SaveTarget) {
  const [busy, setBusy] = useState(false);
  const [problems, setProblems] = useState<readonly string[]>([]);
  return {
    busy,
    problems,
    submit: async () => {
      const found = problemsOf(draft);
      setProblems(found);
      if (found.length > 0) {
        return;
      }
      setBusy(true);
      try {
        await save.onSave(draft);
      } catch (error) {
        setProblems([sentenceOf(error)]);
      } finally {
        setBusy(false);
      }
    },
  };
}

function SaveBar({ draft, save }: { draft: TargetDraft; save: SaveTarget }) {
  const { busy, problems, submit } = useSave(draft, save);
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
 * The Target editor: one section per row of targetSections over one draft.
 * Without `save` it edits but offers no Save.
 */
export function TargetEditor({ initial, save }: { initial: TargetDraft; save?: SaveTarget }) {
  const [draft, setDraft] = useState(initial);
  return (
    <div className="flex max-w-[880px] flex-col">
      {targetSections.map((section) => (
        <Section key={section.id} id={section.id} title={section.label}>
          <section.component draft={draft} onChange={setDraft} />
        </Section>
      ))}
      {save !== undefined && <SaveBar draft={draft} save={save} />}
    </div>
  );
}
