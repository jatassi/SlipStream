import { AlertLine } from "../../../../ui/alert-line";
import { KeyCheck } from "./key-check";
import { NodeAddress } from "./node-address";
import type { AddNodeState } from "./state";
import { Waiting } from "./waiting";
import { type Mint, MintButton, Unshown, Ways } from "./ways";

function Closed({ text, mint }: { text: string; mint: Mint }) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <p className="flex-1 text-ink-2">{text}</p>
      <MintButton {...mint} />
    </div>
  );
}

export type BodyProps = {
  readonly id: string;
  readonly state: AddNodeState;
  readonly controller: string;
  readonly mint: Mint;
};

/** The dialog's content for where the code stands. */
export function Body({ id, state, controller, mint }: BodyProps) {
  switch (state.kind) {
    case "loading": {
      return <Waiting text="Loading the Join Code…" />;
    }
    case "failed": {
      return <AlertLine text={state.sentence} />;
    }
    case "expired": {
      return <Closed text="This Join Code expired before a Node used it." mint={mint} />;
    }
    case "revoked": {
      return <Closed text="This Join Code was revoked, so no Node can use it." mint={mint} />;
    }
    case "pairing": {
      return <Waiting text="A Node used the code. Waiting for it to connect…" />;
    }
    case "paired": {
      return <KeyCheck node={state.node} reached={state.reached} />;
    }
    case "open": {
      return (
        <div className="flex flex-col gap-5">
          {state.code === undefined ? (
            <Unshown left={state.left} mint={mint} />
          ) : (
            <Ways code={state.code} left={state.left} />
          )}
          <Waiting text="Waiting for the new Node to pair…" />
          <NodeAddress id={id} controller={controller} dial={state.dial} />
        </div>
      );
    }
    default: {
      return state satisfies never;
    }
  }
}
