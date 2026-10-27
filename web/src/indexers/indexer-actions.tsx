import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { IndexersIndexer, NodesNode } from "../api/gen/model";
import { testIndexer } from "../api/gen/query/indexers/indexers";
import { countWords } from "../format";
import { Button } from "../ui/button";
import { EditIndexerDialog } from "./edit-indexer";
import { DeleteIndexerDialog } from "./delete-indexer";
import { MoveIndexerDialog } from "./move-indexer";

const items = { one: "item", many: "items" } as const;

type Result = { readonly passed: boolean; readonly words: string };

/** Runs the Indexer's Test; its new status arrives with indexers.status-changed. */
function useTest(indexer: IndexersIndexer) {
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<Result>();
  const run = async () => {
    setBusy(true);
    setResult(undefined);
    try {
      const report = await testIndexer(indexer.id);
      const listed = countWords(report.items, items);
      setResult({ passed: true, words: `${indexer.name} passed the Test and listed ${listed}.` });
    } catch (error) {
      setResult({ passed: false, words: sentenceOf(error) });
    } finally {
      setBusy(false);
    }
  };
  return { busy, result, run };
}

type Open = "edit" | "move" | "delete" | undefined;

function TestResult({ result }: { result: Result | undefined }) {
  if (result === undefined) {
    return null;
  }
  return result.passed ? (
    <output className="text-ink-2">{result.words}</output>
  ) : (
    <p role="alert" className="text-fail-ink">
      {result.words}
    </p>
  );
}

/** Test, Edit, Move and Delete, with the Test's answer under them. */
export function IndexerActions({
  indexer,
  nodes,
}: {
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
}) {
  const [open, setOpen] = useState<Open>();
  const test = useTest(indexer);
  const opener = (dialog: Open) => () => {
    setOpen(dialog);
  };
  const close = (next: boolean) => {
    if (!next) {
      setOpen(undefined);
    }
  };
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-2">
        <Button busy={test.busy} onClick={() => void test.run()}>
          Test
        </Button>
        <Button onClick={opener("edit")}>Edit</Button>
        <Button onClick={opener("move")}>Move</Button>
        <Button look="ghost" onClick={opener("delete")}>
          Delete
        </Button>
      </div>
      <TestResult result={test.result} />
      {open === "edit" && <EditIndexerDialog indexer={indexer} open onOpenChange={close} />}
      {open === "move" && (
        <MoveIndexerDialog indexer={indexer} nodes={nodes} open onOpenChange={close} />
      )}
      {open === "delete" && <DeleteIndexerDialog indexer={indexer} open onOpenChange={close} />}
    </div>
  );
}
