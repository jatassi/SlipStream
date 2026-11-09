import { useId } from "react";

import type { FormatsDraft } from "../api/gen/model";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { dataFont } from "../ui/data";
import { Section } from "../ui/section";
import { Segmented } from "../ui/segmented";
import { TestedName } from "./tested-name";
import { type TesterKind, kindLabels, nameLimit, namesIn, namesProblem } from "./tester-words";
import { type Outcome, type Testing, type WorkRef, useTesting } from "./testing";

function NamesBox({
  kind,
  text,
  problem,
  onText,
}: {
  kind: TesterKind;
  text: string;
  problem: string | undefined;
  onText: (text: string) => void;
}) {
  const id = useId();
  return (
    <div className="flex flex-col gap-1.5">
      <label
        htmlFor={id}
        className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold"
      >
        {kindLabels[kind]}
      </label>
      <textarea
        id={id}
        rows={4}
        spellCheck={false}
        value={text}
        aria-invalid={problem === undefined ? undefined : true}
        aria-describedby={`${id}-message`}
        onChange={(event) => {
          onText(event.target.value);
        }}
        className={`min-h-(--input-height) w-full resize-y rounded-item bg-well px-3 py-2 text-body text-ink-1 ring-1 ring-hairline ring-inset aria-invalid:ring-fail-line in-data-[density=touch]:text-input-touch ${dataFont}`}
      />
      <p
        id={`${id}-message`}
        className={`text-meta in-data-[density=touch]:text-body-touch ${problem === undefined ? "text-ink-3" : "text-fail-ink"}`}
      >
        {problem ?? `One per line, up to ${String(nameLimit)}.`}
      </p>
    </div>
  );
}

function Judging({ work, onClear }: { work: WorkRef; onClear: () => void }) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
      <p className="text-ink-2">Each name is judged against {work.label}.</p>
      <Button look="ghost" size="small" onClick={onClear}>
        Match by title instead
      </Button>
    </div>
  );
}

function Results({ outcome }: { outcome: Outcome }) {
  switch (outcome.state) {
    case "idle":
      return null;
    case "refused":
      return <AlertLine text={outcome.sentence} />;
    case "results":
      return (
        <ol aria-label="Results" className="flex flex-col">
          {outcome.results.map((result, index) => (
            // Two pasted lines can be the same name.
            <TestedName key={`${String(index)} ${result.name}`} result={result} />
          ))}
        </ol>
      );
    default:
      return outcome satisfies never;
  }
}

function Controls({ testing, draft }: { testing: Testing; draft: FormatsDraft | undefined }) {
  const { input, busy, edit, test } = testing;
  const names = namesIn(input.text);
  const problem = namesProblem(names, input.kind);
  return (
    <>
      <div>
        <Segmented
          label="Read each line as"
          options={[
            { value: "release", label: kindLabels.release },
            { value: "file", label: kindLabels.file },
          ]}
          value={input.kind}
          onValueChange={(kind) => {
            edit({ ...input, kind });
          }}
        />
      </div>
      <NamesBox
        kind={input.kind}
        text={input.text}
        problem={problem}
        onText={(text) => {
          edit({ ...input, text });
        }}
      />
      <div>
        <Button
          busy={busy}
          disabled={names.length === 0 || problem !== undefined}
          onClick={() => {
            test(input, draft);
          }}
        >
          Test
        </Button>
      </div>
    </>
  );
}

/**
 * Reads pasted release names or file paths as a search would, one sentence
 * per name, with the facts, formats and per-Target verdicts under Details.
 */
export function Tester({ testing, draft }: { testing: Testing; draft?: FormatsDraft | undefined }) {
  const { input, outcome, edit } = testing;
  return (
    <div className="flex flex-col gap-4">
      {draft !== undefined && (
        <p className="text-ink-2">
          {draft.name} is matched as it stands in the editor, before it&apos;s saved.
        </p>
      )}
      {input.work !== undefined && (
        <Judging
          work={input.work}
          onClear={() => {
            edit({ ...input, work: undefined });
          }}
        />
      )}
      <Controls testing={testing} draft={draft} />
      <Results outcome={outcome} />
    </div>
  );
}

/** The tester at the top of the Custom Formats section. */
export function TesterSection() {
  const testing = useTesting();
  return (
    <Section title="Tester">
      <div className="flex flex-col gap-4">
        <p className="text-ink-2">
          Paste release names or file paths to see what SlipStream reads in each, the Work it
          matches, the Custom Formats it matches and whether a search would grab it.
        </p>
        <Tester testing={testing} />
      </div>
    </Section>
  );
}
