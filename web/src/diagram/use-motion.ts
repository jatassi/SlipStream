import { useInView, useLoop } from "@anatomy/react/live";
import { type RefObject, useEffect, useLayoutEffect, useRef, useSyncExternalStore } from "react";

import { subscribeTraffic, trafficPair } from "../live/traffic";
import type { Landing } from "../routes/_app/nodes/-landing/view";
import type { Drawing } from "./drawing";
import { type Engine, engineOf } from "./live";
import { targetsOf } from "./model";

const KEPT_FRAMES = 600;

function keepFrameTime(ms: number) {
  // oxlint-disable-next-line no-underscore-dangle -- the name the frame budget check reads
  const ticks = (window.__diagramTickMs ??= []);
  ticks.push(ms);
  if (ticks.length > KEPT_FRAMES) {
    ticks.shift();
  }
}

/** When the latest traffic sample arrived, by the page's clock. */
function useArrival(): RefObject<number> {
  const arrived = useRef(0);
  useEffect(() => {
    arrived.current = performance.now();
    return subscribeTraffic(() => {
      arrived.current = performance.now();
    });
  }, []);
  return arrived;
}

/**
 * Moves the drawing: a frame loop while it is on screen and motion is
 * allowed, or one still frame per change under reduced motion. Development
 * builds keep each frame's script time for the frame budget check.
 */
export function useMotion({
  figure,
  svg,
  drawing,
  landing,
  still,
}: {
  figure: RefObject<HTMLElement | null>;
  svg: RefObject<SVGSVGElement | null>;
  drawing: Drawing;
  landing: Landing;
  still: boolean;
}) {
  const engine = useRef<Engine>(undefined);
  const inView = useInView(figure);
  const arrived = useArrival();
  const samples = useSyncExternalStore(subscribeTraffic, trafficPair);

  useLayoutEffect(() => {
    const made = svg.current === null ? undefined : engineOf(svg.current, drawing);
    engine.current = made;
    return () => {
      made?.dispose();
      engine.current = undefined;
    };
  }, [svg, drawing]);

  useEffect(() => {
    const drawn = engine.current;
    if (still && drawn?.drawing === drawing) {
      drawn.still(targetsOf(landing, samples, Number.POSITIVE_INFINITY));
    }
  }, [still, landing, samples, drawing]);

  useLoop((dt) => {
    const started = performance.now();
    engine.current?.frame(targetsOf(landing, trafficPair(), started - arrived.current), dt);
    if (import.meta.env.DEV) {
      keepFrameTime(performance.now() - started);
    }
    return true;
  }, inView && !still);
}
