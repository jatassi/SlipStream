import type { NodesNode } from "../../../../api/gen/model";
import { type Clock, clock as clockWords, sinceWords } from "../../../../format";
import { reachedBy } from "../../../../nodes/dial";
import { Section } from "../../../../ui/section";
import { KeyValues } from "../-facts";
import { Findings, findingsOf } from "../-findings";
import type { Acting } from "../-manage/dialogs";
import { useNodeState } from "../-node";
import { Labelled } from "./labelled";
import { Roles } from "./roles";

type Props = {
  readonly node: NodesNode;
  readonly nodes: readonly NodesNode[];
  readonly clock: Clock;
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
      "Reached by",
      <Labelled
        key="reached"
        text={reachedBy(controller, node)}
        action={{ label: "Change", onClick: change }}
      />,
    ] as const,
  ];
}

/** What the Controller knows of the Node: its name, Roles, key, pairing, and how it is reached. */
export function NodeFacts(props: Props) {
  const { node, nodes, clock, act } = props;
  const state = useNodeState(node);
  const away = state !== "online" && state !== "restarting";
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
            ["Roles", <Roles key="roles" node={node} nodes={nodes} clock={clock} />],
            ["Key", node.key],
            ["Paired", clockWords(node.pairedAt)],
            ...(away ? [["Last seen", sinceWords(node.link.since, clock)] as const] : []),
            ...reached(props),
          ]}
        />
        <Findings findings={findingsOf(node.probes)} label={`What ${node.name} found`} />
      </div>
    </Section>
  );
}
