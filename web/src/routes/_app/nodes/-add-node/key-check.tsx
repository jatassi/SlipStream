import type { NodesNode } from "../../../../api/gen/model";
import { Data } from "../../../../ui/data";
import { Fact, Facts } from "../../../../ui/facts";

/** The new Node's key to compare with the one it shows, and which side dials. */
export function KeyCheck({ node, reached }: { node: NodesNode; reached: string }) {
  return (
    <div className="flex flex-col gap-3">
      <p className="text-ink-1">{node.name} paired.</p>
      <Facts look="sheet">
        <Fact label="Key">
          <Data>{node.key}</Data>
        </Fact>
        <Fact label="Connection">{reached}</Fact>
      </Facts>
      <p className="text-ink-2">
        Check that the key matches the one {node.name} shows, so you know you paired the right
        machine.
      </p>
    </div>
  );
}
