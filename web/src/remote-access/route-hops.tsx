import { CheckmarkCircle02Icon } from "@hugeicons/core-free-icons";
import { Fragment } from "react";

import type { ReachabilityHopView } from "../api/gen/model";
import { Icon } from "../ui/icon";
import { type Signal, signals } from "../ui/signal-chip";
import { signalLook } from "../ui/signal-row";
import { hopNames, results } from "./words";

/**
 * A failing hop's signal: red when a failing check points to a relay, amber
 * when it points to a fix. Signal colour comes only from a failing check.
 */
export function hopSignal(hop: ReachabilityHopView): Exclude<Signal, "ok"> | undefined {
  if (hop.worst !== "fails") {
    return undefined;
  }
  const failing = hop.checks.filter((check) => !check.fact && check.result === "fails");
  return failing.some((check) => check.points === "needs-relay") ? "fail" : "warn";
}

const tile =
  "flex min-h-(--control-height) flex-col items-start justify-center gap-0.5 rounded-item px-3 py-1.5 text-left transition-colors duration-(--duration-control) ease-standard focus-visible:outline-2 focus-visible:outline-offset-2";

function tileLook(hop: ReachabilityHopView, selected: boolean): string {
  const signal = hopSignal(hop);
  if (signal !== undefined) {
    return signalLook[signal].row;
  }
  const ring = selected ? "ring-2 ring-ink-2" : "";
  if (hop.worst === "not-reached") {
    return `border border-dashed border-hairline-strong text-ink-3 ${ring}`;
  }
  return `bg-fill hover:bg-fill-hover ${hop.worst === "not-used" ? "text-ink-3" : "text-ink-1"} ${ring}`;
}

function Mark({ hop }: { hop: ReachabilityHopView }) {
  const signal = hopSignal(hop);
  if (signal !== undefined) {
    return <Icon icon={signals[signal].icon} className="size-3.5 shrink-0" />;
  }
  if (hop.worst === "passes") {
    return <Icon icon={CheckmarkCircle02Icon} className="size-3.5 shrink-0 text-ok" />;
  }
  return null;
}

// The chain runs down in a narrow route and across in a wide one, so a
// connector never ends a wrapped line. 39rem holds an own-proxy route's six
// hops, the widest drawn, so the chain stays across beside the Inspector.
function Connector({ after, to }: { after: ReachabilityHopView; to: ReachabilityHopView }) {
  const signal = hopSignal(after);
  const line =
    to.worst === "not-reached"
      ? "w-0 border-l-2 border-dashed border-hairline-strong @min-[39rem]:h-0 @min-[39rem]:border-t-2 @min-[39rem]:border-l-0"
      : signal === undefined
        ? "w-0.5 bg-hairline-strong @min-[39rem]:h-0.5"
        : `w-[3px] @min-[39rem]:h-[3px] ${signal === "fail" ? "bg-fail" : "bg-warn"}`;
  return (
    <span
      aria-hidden="true"
      className={`ml-5 h-3 shrink-0 @min-[39rem]:ml-0 @min-[39rem]:w-3 @min-[39rem]:self-center ${line}`}
    />
  );
}

export function RouteHops({
  subject,
  hops,
  selected,
  onSelect,
}: {
  subject: string;
  hops: readonly ReachabilityHopView[];
  selected: number | undefined;
  onSelect: (index: number) => void;
}) {
  return (
    <div className="@container">
      <ol
        aria-label={`Hops of ${subject}`}
        className="flex flex-col items-start @min-[39rem]:flex-row @min-[39rem]:flex-wrap @min-[39rem]:items-stretch @min-[39rem]:gap-y-2"
      >
        {hops.map((hop, index) => (
          <Fragment key={hop.kind}>
            {index > 0 && <Connector after={hops[index - 1] ?? hop} to={hop} />}
            <li className="flex">
              <button
                type="button"
                aria-pressed={selected === index}
                aria-label={`${hopNames[hop.kind]}: ${results[hop.worst]}`}
                className={`${tile} ${tileLook(hop, selected === index)}`}
                onClick={() => {
                  onSelect(index);
                }}
              >
                <span className="inline-flex items-center gap-1.5 font-semibold">
                  <Mark hop={hop} />
                  {hopNames[hop.kind]}
                </span>
                {hop.worst === "not-reached" && (
                  <>
                    {" "}
                    <span className="text-meta">not reached</span>
                  </>
                )}
              </button>
            </li>
          </Fragment>
        ))}
      </ol>
    </div>
  );
}
