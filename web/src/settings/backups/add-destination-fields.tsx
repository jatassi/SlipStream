import { useSelector } from "@tanstack/react-form";
import { useState } from "react";

import type { NodesNode } from "../../api/gen/model";
import { Button } from "../../ui/button";
import { useAddDestination } from "./add-destination-form";
import { BackupsFolder, NodePicker } from "./backups-folder";
import { SignalLine } from "../../ui/signal-line";

export function AddFields({ nodes, close }: { nodes: readonly NodesNode[]; close: () => void }) {
  const [nodeId, setNodeId] = useState(() => nodes.find((node) => !node.self)?.id ?? "");
  const node = nodes.find((each) => each.id === nodeId);
  const { form, refusal } = useAddDestination(node, close);
  const path = useSelector(form.store, (state) => state.values.path);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <p>
        SlipStream copies every Backup to a folder on this Node. Pick a Node on another machine, or
        another disk on this one.
      </p>
      <NodePicker
        nodes={nodes}
        nodeId={nodeId}
        onNodeChange={(next) => {
          setNodeId(next);
          form.reset();
        }}
      />
      <form.AppForm>
        {node !== undefined && (
          <form.AppField name="path">{() => <BackupsFolder node={node} />}</form.AppField>
        )}
        {refusal !== undefined && (
          <SignalLine tone="fail" announce>
            {refusal}
          </SignalLine>
        )}
        <div className="flex justify-end gap-2">
          <Button onClick={close}>Cancel</Button>
          {path.trim() !== "" && <form.SubmitButton label="Add" />}
        </div>
      </form.AppForm>
    </form>
  );
}
