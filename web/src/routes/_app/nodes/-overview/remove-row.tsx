import type { NodesNode } from "../../../../api/gen/model";
import { Button } from "../../../../ui/button";
import { Section } from "../../../../ui/section";
import type { Acting } from "../-manage/dialogs";
import { controllerStays } from "../-manage/remove";

/** The danger row: Remove for a Worker, and why the Controller stays. */
export function RemoveRow({ node, act }: { node: NodesNode; act: (acting: Acting) => void }) {
  if (node.self) {
    return (
      <Section title="Remove">
        <p className="max-w-[640px] text-ink-2">{controllerStays(node.name)}</p>
      </Section>
    );
  }
  return (
    <Section title="Remove">
      <div className="flex flex-col items-start gap-3">
        <p className="max-w-[640px] text-ink-2">
          {node.name} leaves the Roster and stops working for this SlipStream. Its files stay on it.
        </p>
        <Button
          look="fail"
          onClick={() => {
            act({ kind: "remove", nodeId: node.id });
          }}
        >
          Remove {node.name}
        </Button>
      </div>
    </Section>
  );
}
