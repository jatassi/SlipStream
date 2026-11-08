import { useState } from "react";

import type { FormatsDraft } from "../api/gen/model";
import { Dialog } from "../ui/dialog";
import type { Problems } from "./draft";
import { Tester } from "./tester";
import { type WorkRef, useTesting } from "./testing";

export type TesterOpening = {
  /** Fills the names and tests them at once; left out, the last names stay. */
  readonly names?: readonly string[];
  readonly work?: WorkRef;
  readonly draft?: FormatsDraft;
};

/**
 * The tester in a dialog over the page that opens it: the format editor's
 * "Try it" with its draft, and a Release's "Open in tester" with its name
 * and Work.
 */
export function useTester(onDraftRefused?: (problems: Problems) => void) {
  const testing = useTesting(onDraftRefused);
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<FormatsDraft>();
  return {
    open: ({ names, work, draft: opened }: TesterOpening) => {
      const input =
        names === undefined
          ? { ...testing.input, work }
          : { text: names.join("\n"), kind: "release" as const, work };
      setDraft(opened);
      setOpen(true);
      testing.test(input, opened);
    },
    dialog: (
      <Dialog title="Tester" size={940} open={open} onOpenChange={setOpen}>
        <Tester testing={testing} draft={draft} />
      </Dialog>
    ),
  };
}
