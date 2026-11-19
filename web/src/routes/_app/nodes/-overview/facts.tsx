import type { NodesNode } from "../../../../api/gen/model";
import { clock as clockWords } from "../../../../format";
import { reachedBy } from "../../../../nodes/dial";
import { Data } from "../../../../ui/data";
import { Fact, Facts } from "../../../../ui/facts";
import { Section } from "../../../../ui/section";
import { Findings, findingsOf } from "../-findings";
import type { Acting } from "../-manage/dialogs";

/** What the Controller knows of the Node: its name, key, pairing, and how it is reached. */
export function NodeFacts({
  node,
  nodes,
  act,
}: {
  readonly node: NodesNode;
  readonly nodes: readonly NodesNode[];
  readonly act: (acting: Acting) => void;
}) {
  const controller = nodes.find(({ self }) => self)?.name ?? "the Controller";
  return (
    <Section title="Overview">
      <div className="flex flex-col gap-3">
        <Facts look="sheet">
          <Fact
            label="Name"
            action={{
              label: "Rename",
              onClick: () => {
                act({ kind: "rename", nodeId: node.id });
              },
            }}
          >
            {node.name}
          </Fact>
          <Fact label="Key">
            <Data>{node.key}</Data>
          </Fact>
          <Fact label="Paired">
            <Data>{clockWords(node.pairedAt)}</Data>
          </Fact>
          {!node.self && (
            <Fact
              label="Connection"
              action={{
                label: "Change",
                onClick: () => {
                  act({ kind: "reach", nodeId: node.id });
                },
              }}
            >
              {reachedBy(controller, node)}
            </Fact>
          )}
        </Facts>
        <Findings findings={findingsOf(node.probes)} label={`What ${node.name} found`} />
      </div>
    </Section>
  );
}
