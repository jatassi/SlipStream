import { useState } from "react";

import { DialogActions } from "../account/dialog-actions";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Spinner } from "../ui/spinner";
import { Code } from "./join-code";
import { useJoinCode } from "./use-join-code";
import { SignalLine } from "../ui/signal-line";

function Body({ node }: { node: string }) {
  const minted = useJoinCode();
  switch (minted.kind) {
    case "minting":
      return (
        <p className="flex items-center gap-2 text-ink-2">
          <Spinner />
          Creating a Join Code…
        </p>
      );
    case "failed":
      return (
        <SignalLine tone="fail" announce>
          {minted.sentence}
        </SignalLine>
      );
    case "minted":
      return <Code code={minted.code} node={node} />;
    default:
      return minted satisfies never;
  }
}

/**
 * A Join Code for a Node to pair with this Controller again. The Node keeps
 * its torrents and restarts once it takes the code.
 */
export function PairDialog({
  title,
  node,
  open,
  onOpenChange,
}: {
  title: string;
  /** The Node as the lead names it, such as vps-relay or the Node at 203.0.113.7. */
  node: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title={title} size={680} open={open} onOpenChange={onOpenChange}>
      <div className="flex flex-col gap-5">
        <p className="text-ink-2">
          Give {node} this Join Code. It pairs with this Controller, keeps its torrents and
          restarts.
        </p>
        {open && <Body node={node} />}
        <DialogActions>
          <Button
            onClick={() => {
              onOpenChange(false);
            }}
          >
            Close
          </Button>
        </DialogActions>
      </div>
    </Dialog>
  );
}

export function PairButton({ title, node }: { title: string; node: string }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setOpen(true);
        }}
      >
        New Join Code
      </Button>
      <PairDialog title={title} node={node} open={open} onOpenChange={setOpen} />
    </>
  );
}
