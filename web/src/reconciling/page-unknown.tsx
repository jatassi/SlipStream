import type { ReconcilingUnknown } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { PairButton } from "./pair-dialog";
import { unknownLine } from "./words";

function UnknownRow({ node, clock }: { node: ReconcilingUnknown; clock: Clock }) {
  return (
    <ListRow
      title={node.address}
      wrap
      detail={
        <div className="flex flex-col gap-2 pb-1 text-ink-2">
          <p>{unknownLine(node, clock)}</p>
          <Details>
            <p>
              Its node key&apos;s fingerprint is <Data>{node.fingerprint}</Data>.
            </p>
          </Details>
          <div className="pt-1">
            <PairButton
              title={`Give the Node at ${node.address} a Join Code`}
              node={`the Node at ${node.address}`}
            />
          </div>
        </div>
      }
    />
  );
}

export function UnknownSection({
  unknown,
  clock,
}: {
  unknown: readonly ReconcilingUnknown[];
  clock: Clock;
}) {
  if (unknown.length === 0) {
    return null;
  }
  return (
    <Section title="Nodes not in this Backup">
      <List label="Nodes not in this Backup">
        {unknown.map((node) => (
          <UnknownRow key={node.fingerprint} node={node} clock={clock} />
        ))}
      </List>
    </Section>
  );
}
