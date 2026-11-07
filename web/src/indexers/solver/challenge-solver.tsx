import type { IndexersChallengeSolver, NodesNode } from "../../api/gen/model";
import { useGetChallengeSolverSuspense } from "../../api/gen/query/indexers/indexers";
import { ReadOnlyField } from "../../ui/read-only-field";
import { Section } from "../../ui/section";
import { SolverForm } from "./solver-form";

function AwaySolver({
  node,
  away,
  solver,
}: {
  node: NodesNode;
  away: string;
  solver: IndexersChallengeSolver;
}) {
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-1">
        {`${node.name} is ${away}. Saving tests the solver from ${node.name}, so its Challenge solver can change once it's back.`}
      </p>
      {solver.address === null ? (
        <p className="text-ink-3">{node.name} has no Challenge solver.</p>
      ) : (
        <>
          <ReadOnlyField label="Address" value={solver.address} />
          <ReadOnlyField label="Timeout" value={`${solver.timeout} s`} />
        </>
      )}
    </div>
  );
}

/** A Node's Challenge solver: a form while the Node is online, read-only while it's away. */
export function ChallengeSolver({ node, away }: { node: NodesNode; away: string | null }) {
  const solver = useGetChallengeSolverSuspense(node.id).data;
  return (
    <Section title="Challenge solver">
      <p className="mb-4 text-ink-2">
        Some sites sit behind a browser challenge. A solver clears it for each request {node.name}{" "}
        sends them.
      </p>
      {away === null ? (
        <SolverForm key={`${solver.address}:${solver.timeout}`} node={node} solver={solver} />
      ) : (
        <AwaySolver node={node} away={away} solver={solver} />
      )}
    </Section>
  );
}
