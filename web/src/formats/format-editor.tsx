import { useState } from "react";

import { fieldErrors, isApiError, isProblem, sentenceOf } from "../api/errors";
import type { FormatsSchema } from "../api/gen/model";
import { DraftField } from "../targets/field";
import { Section } from "../ui/section";
import { Switch } from "../ui/switch";
import { ConditionsSection, type Slow } from "./conditions-section";
import { type FormatDraft, type Problems, problemsOf } from "./draft";
import { EditorActions } from "./editor-actions";

function refusedOf(error: unknown): Problems {
  if (isProblem(error, "format_name_taken")) {
    return { name: "Another Custom Format has this name. Choose another name." };
  }
  if (isApiError(error) && error.errors.length > 0) {
    return fieldErrors(error);
  }
  return { save: sentenceOf(error) };
}

const placedFields = ["kind", "fields.value", "fields.min", "fields.max"];

/** The problems no field of the draft shows, which sit under the save button. */
function unplaced(problems: Problems, draft: FormatDraft): string[] {
  const placed = new Set([
    "name",
    "conditions",
    ...draft.conditions.flatMap((_, index) =>
      placedFields.map((field) => `conditions[${String(index)}].${field}`),
    ),
  ]);
  return Object.entries(problems).flatMap(([key, text]) => (placed.has(key) ? [] : [text]));
}

function NameFields({
  draft,
  problem,
  onChange,
}: {
  draft: FormatDraft;
  problem: string | undefined;
  onChange: (next: FormatDraft) => void;
}) {
  return (
    <div className="flex max-w-[400px] flex-col gap-3">
      <DraftField
        label="Name"
        value={draft.name}
        error={problem}
        onChange={(name) => {
          onChange({ ...draft, name });
        }}
      />
      <div className="flex flex-col">
        <Switch
          label="Include when renaming"
          checked={draft.includeWhenRenaming}
          onCheckedChange={(includeWhenRenaming) => {
            onChange({ ...draft, includeWhenRenaming });
          }}
        />
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          The Custom Formats naming token lists this format when a file matches it.
        </p>
      </div>
    </div>
  );
}

function useSave(draft: FormatDraft, onSave: (draft: FormatDraft) => Promise<unknown>) {
  const [problems, setProblems] = useState<Problems>({});
  const [busy, setBusy] = useState(false);
  const submit = async () => {
    const found = problemsOf(draft);
    setProblems(found);
    if (Object.keys(found).length > 0) {
      return;
    }
    setBusy(true);
    try {
      await onSave(draft);
    } catch (error) {
      setProblems(refusedOf(error));
    } finally {
      setBusy(false);
    }
  };
  return {
    problems,
    busy,
    submit: () => void submit(),
    refuse: setProblems,
    clear: () => {
      setProblems({});
    },
  };
}

export function FormatEditor({
  initial,
  formatId,
  schema,
  slow,
  label,
  onSave,
}: {
  initial: FormatDraft;
  /** The saved format the draft edits, whose app a tried draft is read in. */
  formatId?: string;
  schema: FormatsSchema;
  slow?: Slow;
  label: string;
  onSave: (draft: FormatDraft) => Promise<unknown>;
}) {
  const [draft, setDraft] = useState(initial);
  const { problems, busy, submit, refuse, clear } = useSave(draft, onSave);
  const edit = (next: FormatDraft) => {
    setDraft(next);
    clear();
  };
  return (
    <div className="flex max-w-[880px] flex-col">
      <Section id="name" title="Name">
        <NameFields draft={draft} problem={problems["name"]} onChange={edit} />
      </Section>
      <Section id="conditions" title="Conditions">
        <ConditionsSection
          draft={draft}
          schema={schema}
          slow={slow}
          problems={problems}
          onChange={edit}
        />
      </Section>
      <EditorActions
        draft={draft}
        formatId={formatId}
        unplaced={unplaced(problems, draft)}
        label={label}
        busy={busy}
        onSave={submit}
        onRefused={refuse}
      />
    </div>
  );
}
