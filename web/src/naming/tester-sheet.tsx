import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { WorksContentType } from "../api/gen/model";
import { NamesBox } from "../formats/tester";
import { namesIn, namesProblem } from "../formats/tester-words";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import type { Draft } from "./card";
import { testNames } from "./test-names";
import { Results, type Shown } from "./tested-files";

/**
 * The Naming section's file name tester: for each pasted name, the Work it
 * matches, what SlipStream reads in it and the path an import would give it
 * under its Content Type's card, unsaved changes included.
 */
export function TesterSheet({
  types,
  draftOf,
}: {
  types: readonly WorksContentType[];
  draftOf: (type: string) => Draft;
}) {
  const [text, setText] = useState("");
  const [shown, setShown] = useState<Shown>({ state: "idle" });
  const [busy, setBusy] = useState(false);
  const names = namesIn(text);
  const problem = namesProblem(names, "file");
  const test = async () => {
    setBusy(true);
    try {
      setShown({ state: "results", files: await testNames(names, types, draftOf) });
    } catch (error) {
      setShown({ state: "refused", sentence: sentenceOf(error) });
    } finally {
      setBusy(false);
    }
  };
  return (
    <Dialog title="Test file names" size={940} trigger={<Button>Test file names</Button>}>
      <div className="flex flex-col gap-4">
        <p className="text-ink-2">
          Paste file names to see the Work each one matches, what SlipStream reads in it and the
          path it would be imported at. Each path follows its Content Type&apos;s card as it stands,
          unsaved changes included.
        </p>
        <NamesBox kind="file" text={text} problem={problem} onText={setText} />
        <div>
          <Button
            busy={busy}
            disabled={names.length === 0 || problem !== undefined}
            onClick={() => void test()}
          >
            Test
          </Button>
        </div>
        <Results shown={shown} />
      </div>
    </Dialog>
  );
}
