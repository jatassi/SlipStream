import { useSelector } from "@tanstack/react-form";
import { useState } from "react";

import type { NodesNode } from "../../api/gen/model";
import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";
import { useAddDestination } from "./add-destination-form";
import { BackupsFolder, NodePicker } from "./backups-folder";

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
        SlipStream copies every Backup to the folder you pick. Pick one on another machine, or on
        another disk of the Controller's machine.
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
        {refusal !== undefined && <AlertLine text={refusal} />}
        <div className="flex justify-end gap-2">
          <Button onClick={close}>Cancel</Button>
          {path.trim() !== "" && <form.SubmitButton label="Add" />}
        </div>
      </form.AppForm>
    </form>
  );
}
