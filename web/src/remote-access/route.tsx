import type { ReachabilityRoute } from "../api/gen/model";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { Defined } from "../ui/defined";
import { VerdictChips } from "./chips";
import { RouteHops } from "./route-hops";
import { RouteMenu } from "./route-menu";
import { routeKinds } from "./words";

export const routeAnchor = (exposureId: string) => `route-${exposureId}`;

function Heading({ route }: { route: ReachabilityRoute }) {
  const kind =
    route.route === undefined || route.route === "unplaced" ? undefined : routeKinds[route.route];
  return (
    <div className="flex flex-wrap items-start gap-x-3 gap-y-1.5">
      <div className="flex min-w-0 flex-[1_1_20rem] flex-wrap items-center gap-x-3 gap-y-1.5">
        <h3 className="font-semibold text-ink-1">{route.exposure.subject}</h3>
        <Data className="min-w-0 truncate text-ink-2">{route.exposure.public}</Data>
        <span className="flex items-center gap-x-3 whitespace-nowrap">
          {route.families !== undefined && <Chip look="outline">{route.families}</Chip>}
          {kind !== undefined && (
            <Defined term={kind}>
              {(word) => <span className="text-meta text-ink-3">{word}</span>}
            </Defined>
          )}
        </span>
      </div>
      <div className="flex shrink-0 items-center gap-3">
        <VerdictChips route={route} />
        <RouteMenu exposure={route.exposure} />
      </div>
    </div>
  );
}

/**
 * One wanted thing's route: one line once it works and was tested, and
 * otherwise drawn hop by hop so the hop where it stops shows.
 */
export function RouteCard({
  route,
  selectedHop,
  onSelectHop,
}: {
  route: ReachabilityRoute;
  selectedHop: number | undefined;
  onSelectHop: (index: number) => void;
}) {
  return (
    <section
      id={routeAnchor(route.exposure.id)}
      aria-label={route.exposure.subject}
      className="flex scroll-mt-20 flex-col gap-3 rounded-card bg-glass px-4 py-3 ring-1 ring-hairline ring-inset"
    >
      <Heading route={route} />
      {!route.checked && <p className="text-ink-2">Not checked yet.</p>}
      {route.skipped !== null && <p className="text-ink-2">{route.skipped}</p>}
      {route.checked && !route.folded && (
        <>
          <RouteHops
            subject={route.exposure.subject}
            hops={route.hops}
            selected={selectedHop}
            onSelect={onSelectHop}
          />
          {route.facts.map((fact) => (
            <p key={fact.id} className="text-ink-2">
              {fact.text}
            </p>
          ))}
        </>
      )}
      {route.testedBy !== undefined && <p className="text-meta text-ink-3">{route.testedBy}</p>}
    </section>
  );
}
