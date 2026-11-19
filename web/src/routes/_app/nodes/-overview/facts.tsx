import type { NodesNode } from "../../../../api/gen/model";
import { clock as clockWords } from "../../../../format";
import { reachedBy } from "../../../../nodes/dial";
import { Section } from "../../../../ui/section";
import { KeyValues } from "../-facts";
import { Findings, findingsOf } from "../-findings";
import type { Acting } from "../-manage/dialogs";
import { Labelled } from "./labelled";

type Props = {
  readonly node: NodesNode;
  readonly nodes: readonly NodesNode[];
  readonly act: (acting: Acting) => void;
};

function reached({ node, nodes, act }: Props) {
  if (node.self) {
    return [];
  }
  const controller = nodes.find(({ self }) => self)?.name ?? "the Controller";
  const change = () => {
    act({ kind: "reach", nodeId: node.id });
  };
  return [
    [
      "Connection",
      <Labelled
        key="reached"
        text={reachedBy(controller, node)}
        action={{ label: "Change", onClick: change }}
      />,
    ] as const,
  ];
}

/** What the Controller knows of the Node: its name, key, pairing, and how it is reached. */
export function NodeFacts(props: Props) {
  const { node, act } = props;
  const rename = () => {
    act({ kind: "rename", nodeId: node.id });
  };
  return (
    <Section title="Overview">
      <div className="flex flex-col gap-3">
        <KeyValues
          rows={[
            [
              "Name",
              <Labelled
                key="name"
                text={node.name}
                action={{ label: "Rename", onClick: rename }}
              />,
            ],
            ["Key", node.key],
            ["Paired", clockWords(node.pairedAt)],
            ...reached(props),
          ]}
        />
        <Findings findings={findingsOf(node.probes)} label={`What ${node.name} found`} />
      </div>
    </Section>
  );
}
