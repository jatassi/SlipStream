import { useLocation } from "@tanstack/react-router";
import { useState } from "react";

import type { IndexersIndexer, NodesNode } from "../api/gen/model";
import { Button } from "../ui/button";
import { EditIndexerDialog } from "./edit-indexer";
import { DeleteIndexerDialog } from "./delete-indexer";
import { CaptchaField, CodeField } from "./fields/captcha";
import { MoveIndexerDialog } from "./move-indexer";
import { type TestForm, type TestResult, useIndexerTest } from "./test-indexer";
import { nodeName } from "./words";

type Open = "edit" | "move" | "delete" | undefined;

function Result({ result }: { result: TestResult | undefined }) {
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

function Buttons({
  form,
  onOpen,
  onCode,
}: {
  form: TestForm;
  onOpen: (dialog: Open) => void;
  onCode: (() => void) | undefined;
}) {
  return (
    <div className="flex flex-wrap gap-2">
      {onCode !== undefined && (
        <Button look="fail" onClick={onCode}>
          Type code
        </Button>
      )}
      <form.Subscribe selector={(state) => state.isSubmitting}>
        {(testing) => (
          <Button type="submit" busy={testing}>
            Test
          </Button>
        )}
      </form.Subscribe>
      <Button
        onClick={() => {
          onOpen("edit");
        }}
      >
        Edit
      </Button>
      <Button
        onClick={() => {
          onOpen("move");
        }}
      >
        Move
      </Button>
      <Button
        look="ghost"
        onClick={() => {
          onOpen("delete");
        }}
      >
        Delete
      </Button>
    </div>
  );
}

/**
 * Test with the buttons that open the dialogs, then the step a sign-in
 * needs (the captcha a failed Test hands back, or the two-factor code) and
 * the Test's answer. A stop for a code offers Type code, which opens its
 * step, and the page opened at #code starts on it.
 */
function TestPanel({
  indexer,
  home,
  onOpen,
}: {
  indexer: IndexersIndexer;
  home: string;
  onOpen: (dialog: Open) => void;
}) {
  const codeAsked = useLocation({ select: (location) => location.hash === "code" });
  const { form, step, result, reload, askCode } = useIndexerTest(indexer, codeAsked);
  const offersCode = indexer.status.cause === "code_needed" && step.kind !== "code";
  return (
    <form
      noValidate
      className="flex flex-col gap-3"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <Buttons form={form} onOpen={onOpen} onCode={offersCode ? askCode : undefined} />
        {step.kind === "code" && (
          <div id="code" className="scroll-mt-20">
            <CodeField form={form} fields={{ code: "code" }} name={indexer.name} />
          </div>
        )}
        {(step.kind === "shown" || step.kind === "expired") && (
          <CaptchaField
            form={form}
            fields={{ captcha: "captcha" }}
            name={indexer.name}
            home={home}
            step={step}
            onReload={reload}
          />
        )}
        <Result result={result} />
      </form.AppForm>
    </form>
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
  const close = (next: boolean) => {
    if (!next) {
      setOpen(undefined);
    }
  };
  return (
    <div>
      <TestPanel indexer={indexer} home={nodeName(nodes, indexer.homeNodeId)} onOpen={setOpen} />
      {open === "edit" && <EditIndexerDialog indexer={indexer} open onOpenChange={close} />}
      {open === "move" && (
        <MoveIndexerDialog indexer={indexer} nodes={nodes} open onOpenChange={close} />
      )}
      {open === "delete" && <DeleteIndexerDialog indexer={indexer} open onOpenChange={close} />}
    </div>
  );
}
