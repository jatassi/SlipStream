import type { IndexersChallengeSolver, NodesNode } from "../../api/gen/model";
import { useGetChallengeSolverSuspense } from "../../api/gen/query/indexers/indexers";
import { type Away, awayWords } from "../../nodes/standing";
import { ReadOnlyField } from "../../ui/read-only-field";
import { Section } from "../../ui/section";
import { SolverForm } from "./solver-form";

function AwaySolver({
  node,
  away,
  solver,
}: {
  node: NodesNode;
  away: Away;
  solver: IndexersChallengeSolver;
}) {
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-1">
        {`${node.name} is ${awayWords[away]}. You can change its Challenge solver once it's back, because saving tests the solver from ${node.name}.`}
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
export function ChallengeSolver({ node }: { node: NodesNode }) {
  const solver = useGetChallengeSolverSuspense(node.id).data;
  return (
    <Section title="Challenge solver">
      <p className="mb-4 text-ink-2">
        Some sites sit behind a browser challenge. A solver clears it for each request {node.name}{" "}
        sends them.
      </p>
      {node.standing === "in-step" ? (
        <SolverForm key={`${solver.address}:${solver.timeout}`} node={node} solver={solver} />
      ) : (
        <AwaySolver node={node} away={node.standing} solver={solver} />
      )}
    </Section>
  );
}
