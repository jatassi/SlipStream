import type { NodesNode } from "../../../../api/gen/model";
import { KeyValues } from "../-facts";

/** The new Node's key to compare with the one it shows, and which side dials. */
export function KeyCheck({ node, reached }: { node: NodesNode; reached: string }) {
  return (
    <div className="flex flex-col gap-3">
      <p className="text-ink-1">{node.name} paired.</p>
      <KeyValues
        rows={[
          ["Key", node.key],
          ["Reached by", reached],
        ]}
      />
      <p className="text-ink-2">
        Check that the key matches the one {node.name} shows, so you know you paired the right
        machine.
      </p>
    </div>
  );
}
