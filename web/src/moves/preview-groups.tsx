import type { MovesPreviewUnit, MovesPreviewUnitVerdict } from "../api/gen/model";
import { VirtualRows } from "../ui/virtual-rows";
import { VerdictLine } from "./verdict-line";

const order = ["red", "amber", "green"] as const satisfies readonly MovesPreviewUnitVerdict[];

const headings: { readonly [V in MovesPreviewUnitVerdict]: string } = {
  red: "Can't move",
  amber: "Moves, with a note",
  green: "Moves",
};

// Past this many lines a group draws only the ones in view.
const drawnWhole = 200;
const lineHeight = 28;

function Lines({
  verdict,
  units,
}: {
  verdict: MovesPreviewUnitVerdict;
  units: readonly MovesPreviewUnit[];
}) {
  if (units.length <= drawnWhole) {
    return (
      <ul className="flex flex-col gap-1.5">
        {units.map((unit) => (
          <li key={unit.key}>
            <VerdictLine unit={unit} />
          </li>
        ))}
      </ul>
    );
  }
  return (
    <div className="flex h-72 flex-col">
      <VirtualRows
        label={headings[verdict]}
        rows={units}
        rowKey={({ key }) => key}
        estimate={lineHeight}
        renderRow={(unit) => <VerdictLine unit={unit} />}
      />
    </div>
  );
}

/** A preview's units grouped by verdict, the ones that stay first, one line each. */
export function PreviewGroups({ units }: { units: readonly MovesPreviewUnit[] }) {
  return (
    <div className="flex flex-col gap-4">
      {order.map((verdict) => {
        const shown = units.filter((unit) => unit.verdict === verdict);
        return (
          shown.length > 0 && (
            <section key={verdict} aria-label={headings[verdict]} className="flex flex-col gap-2">
              <h3 className="text-card-title font-semibold text-ink-1">
                {headings[verdict]} <span className="font-normal text-ink-3">{shown.length}</span>
              </h3>
              <Lines verdict={verdict} units={shown} />
            </section>
          )
        );
      })}
    </div>
  );
}
