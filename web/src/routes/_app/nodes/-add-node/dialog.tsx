import { useState } from "react";

import type { NodesJoinCode, NodesNode } from "../../../../api/gen/model";
import { AlertLine } from "../../../../ui/alert-line";
import { Button } from "../../../../ui/button";
import { Dialog } from "../../../../ui/dialog";
import { Body } from "./body";
import { OpenCodes } from "./open-codes";
import { RolesStep } from "./roles-step";
import { useCode } from "./use-code";
import type { Mint } from "./ways";

type Props = {
  readonly id: string;
  readonly minted: NodesJoinCode | undefined;
  readonly mint: Mint;
  readonly failure: string | undefined;
  readonly onClose: () => void;
  readonly onAdded: (node: NodesNode) => void;
};

function Following({ id, minted, mint, failure, onClose, onAdded }: Props) {
  const { state, codes, nodes } = useCode(id, minted);
  const [step, setStep] = useState<"follow" | "roles">("follow");
  const controller = nodes.find(({ self }) => self)?.name ?? "the Controller";
  if (state.kind === "paired" && step === "roles") {
    return (
      <RolesStep
        node={state.node}
        onClose={onClose}
        onApplied={() => {
          onAdded(state.node);
        }}
      />
    );
  }
  const paired = state.kind === "paired";
  return (
    <div className="flex flex-col gap-6">
      {failure !== undefined && <AlertLine text={failure} />}
      <div aria-live="polite">
        <Body id={id} state={state} controller={controller} mint={mint} />
      </div>
      {!paired && <OpenCodes codes={codes} current={id} />}
      <div className="flex justify-end gap-2">
        <Button look={paired ? "default" : "ghost"} onClick={onClose}>
          Close
        </Button>
        {paired && (
          <Button
            look="primary"
            onClick={() => {
              setStep("roles");
            }}
          >
            Choose its Roles
          </Button>
        )}
      </div>
    </div>
  );
}

function Refused({ failure, onClose }: { failure: string | undefined; onClose: () => void }) {
  return (
    <div className="flex flex-col gap-6">
      {failure !== undefined && <AlertLine text={failure} />}
      <div className="flex justify-end">
        <Button onClick={onClose}>Close</Button>
      </div>
    </div>
  );
}

/**
 * Add a Node: a Join Code to use on the new Node, followed until a Node
 * pairs with it, then its key and its Roles.
 */
export function AddNodeDialog({ id, ...props }: Omit<Props, "id"> & { id: string | undefined }) {
  return (
    <Dialog
      title="Add a Node"
      size={680}
      open
      onOpenChange={(open) => {
        if (!open) {
          props.onClose();
        }
      }}
    >
      {id === undefined ? (
        <Refused failure={props.failure} onClose={props.onClose} />
      ) : (
        <Following key={id} id={id} {...props} />
      )}
    </Dialog>
  );
}
