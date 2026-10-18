import { useState } from "react";
import { z } from "zod";

import { sentenceOf } from "../../../../api/errors";
import type { NodesNode } from "../../../../api/gen/model";
import { setNodeDial } from "../../../../api/gen/query/nodes/nodes";
import { Button } from "../../../../ui/button";
import { Dialog } from "../../../../ui/dialog";
import { submitErrors, useAppForm } from "../../../../ui/form";

const addressForm = z.object({ nodeAddress: z.string().trim().min(1, "Paste the Node address") });

type Props = {
  readonly node: NodesNode;
  readonly controller: string;
  readonly onClose: () => void;
  readonly onChanged: () => Promise<void>;
};

function AddressForm({ node, controller, onChanged }: Omit<Props, "onClose">) {
  const form = useAppForm({
    defaultValues: { nodeAddress: "" },
    validators: { onSubmit: addressForm },
    onSubmit: async ({ value, formApi }) => {
      try {
        await setNodeDial(node.id, addressForm.parse(value));
        await onChanged();
      } catch (error) {
        formApi.setErrorMap({
          onSubmit: submitErrors(error, {
            node_address_damaged: "nodeAddress",
            node_key_mismatch: "nodeAddress",
          }),
        });
      }
    },
  });
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
        <form.AppField name="nodeAddress">
          {(field) => (
            <field.TextField
              label="Node address"
              autoComplete="off"
              spellCheck={false}
              description={`Run slipstream node-address on ${node.name} and paste what it prints. It starts with SLN1.`}
            />
          )}
        </form.AppField>
        <form.FormAlert />
        <div className="flex justify-end">
          <form.SubmitButton label={`${controller} dials it`} />
        </div>
      </form.AppForm>
    </form>
  );
}

function StopDialing({ node, controller, onChanged }: Omit<Props, "onClose">) {
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  return (
    <div className="flex flex-col items-start gap-2">
      <Button
        busy={busy}
        onClick={() => {
          setBusy(true);
          setFailure(undefined);
          setNodeDial(node.id, { addresses: [] })
            .then(onChanged)
            .catch((error: unknown) => {
              setFailure(sentenceOf(error));
            })
            .finally(() => {
              setBusy(false);
            });
        }}
      >
        Stop dialing
      </Button>
      <p className="text-meta text-ink-3">
        {node.name} dials {controller} itself from then on.
      </p>
      {failure !== undefined && (
        <p role="alert" className="text-fail-ink">
          {failure}
        </p>
      )}
    </div>
  );
}

/**
 * How the Controller reaches a Worker: dial it at the Node address it prints,
 * after it moved, or stop dialing it so it dials the Controller.
 */
export function ReachDialog({ node, controller, onClose, onChanged }: Props) {
  const dials = node.dial.controllerDials;
  return (
    <Dialog
      title={`How ${controller} reaches ${node.name}`}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <div className="flex flex-col gap-6">
        <p className="text-ink-2">
          {dials.length === 0
            ? `${node.name} dials ${controller}. If ${node.name} can't reach ${controller}, ${controller} can dial it instead.`
            : `${controller} dials ${node.name} at ${dials.join(", ")}.`}
        </p>
        <AddressForm node={node} controller={controller} onChanged={onChanged} />
        {dials.length > 0 && (
          <StopDialing node={node} controller={controller} onChanged={onChanged} />
        )}
      </div>
    </Dialog>
  );
}
