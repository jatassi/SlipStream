import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { type FormatDraft, type Problems, bodyOf, problemsOf } from "./draft";
import { useTester } from "./tester-dialog";

/** Tries the draft as it stands, under the name a new format will take when it has none yet. */
function useTryIt(draft: FormatDraft, formatId: string | undefined, refuse: (p: Problems) => void) {
  const tester = useTester(refuse);
  return {
    dialog: tester.dialog,
    run: () => {
      const { name: _, ...found } = problemsOf(draft);
      refuse(found);
      if (Object.keys(found).length > 0) {
        return;
      }
      const body = bodyOf(draft);
      tester.open({
        draft: {
          ...body,
          name: body.name === "" ? "New Custom Format" : body.name,
          ...(formatId !== undefined && { formatId }),
        },
      });
    },
  };
}

export function EditorActions({
  draft,
  formatId,
  unplaced,
  label,
  busy,
  onSave,
  onRefused,
}: {
  draft: FormatDraft;
  formatId: string | undefined;
  /** The problems no field of the draft shows. */
  unplaced: readonly string[];
  label: string;
  busy: boolean;
  onSave: () => void;
  onRefused: (problems: Problems) => void;
}) {
  const tryIt = useTryIt(draft, formatId, onRefused);
  return (
    <div className="mt-8 flex flex-col items-start gap-3">
      {unplaced.map((text) => (
        <AlertLine key={text} text={text} />
      ))}
      <div className="flex flex-wrap gap-2">
        <Button look="primary" busy={busy} onClick={onSave}>
          {label}
        </Button>
        <Button onClick={tryIt.run}>Try it</Button>
      </div>
      {tryIt.dialog}
    </div>
  );
}
