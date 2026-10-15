import "../../../../brand/startup/stage.css";
import "./startup.css";

import { useEffect, useEffectEvent, useRef, useState, useSyncExternalStore } from "react";
import { flushSync } from "react-dom";

import {
  DURATION,
  PHASES,
  STILL,
  build,
  liftMark,
  render,
} from "../../../../brand/startup/sequence.js";
import { readPref, writePref } from "../prefs";

export const markTransitionName = "slipstream-mark";

export type HandOff = "view-transition" | "fade";

const reducedMotion = "(prefers-reduced-motion: reduce)";
const stillSeconds = 0.6;

let markLanding = false;
const landingListeners = new Set<() => void>();

function setMarkLanding(landing: boolean) {
  markLanding = landing;
  for (const listener of landingListeners) {
    listener();
  }
}

// The mark's name must be on one element at a time, so the element it settles
// into carries the name only while the hand-off runs.
export function useMarkTransitionName(): string | undefined {
  return useSyncExternalStore(
    (listener) => {
      landingListeners.add(listener);
      return () => {
        landingListeners.delete(listener);
      };
    },
    () => (markLanding ? markTransitionName : undefined),
  );
}

type Play = {
  stage: HTMLElement;
  signal: AbortSignal;
  stopAt: (reduced: boolean) => number;
  end: (reduced: boolean) => void;
};

// play draws the sequence until it reaches stopAt or any key, click or tap
// skips it, then calls end on the finished frame.
async function play({ stage, signal, stopAt, end }: Play): Promise<void> {
  const listening = new AbortController();
  let frame = 0;
  signal.addEventListener("abort", () => {
    listening.abort();
    cancelAnimationFrame(frame);
  });
  let skipped = false;
  const skip = () => {
    skipped = true;
  };
  window.addEventListener("keydown", skip, { capture: true, signal: listening.signal });
  window.addEventListener("pointerdown", skip, { capture: true, signal: listening.signal });

  await build(stage);
  if (signal.aborted) {
    return;
  }
  const reduced = matchMedia(reducedMotion).matches;
  render(reduced ? STILL : 0, stage);
  let start: number | undefined;
  const tick = (now: number) => {
    start ??= now;
    const t = (now - start) / 1000;
    const stop = stopAt(reduced);
    if (skipped || t >= stop) {
      listening.abort();
      render(skipped || reduced ? STILL : stop, stage);
      end(reduced);
      return;
    }
    if (!reduced) {
      render(t, stage);
    }
    frame = requestAnimationFrame(tick);
  };
  frame = requestAnimationFrame(tick);
}

const stopTime = (handOff: HandOff, reduced: boolean): number => {
  if (reduced) {
    return stillSeconds;
  }
  return handOff === "fade" ? DURATION : PHASES.exit[0];
};

// settleMark runs a view transition that moves the lockup's mark into the
// element named for it as the layer goes.
function settleMark(stage: HTMLElement, onGone: () => void) {
  liftMark(stage).style.viewTransitionName = markTransitionName;
  const transition = document.startViewTransition({
    update: () => {
      flushSync(() => {
        onGone();
        setMarkLanding(true);
      });
    },
    types: ["startup"],
  });
  void transition.finished.finally(() => {
    setMarkLanding(false);
  });
}

export function Startup({ handOff }: { handOff: HandOff }) {
  const [shown, setShown] = useState(() => !readPref("startupOff") && !readPref("startupPlayed"));
  return shown ? (
    <Layer
      handOff={handOff}
      onGone={() => {
        setShown(false);
      }}
    />
  ) : null;
}

function Layer({ handOff, onGone }: { handOff: HandOff; onGone: () => void }) {
  const stageRef = useRef<HTMLDivElement>(null);
  const [fading, setFading] = useState(false);

  const stopAt = useEffectEvent((reduced: boolean) => stopTime(handOff, reduced));
  const end = useEffectEvent((stage: HTMLElement, reduced: boolean) => {
    if (reduced || handOff === "fade") {
      setFading(true);
    } else {
      settleMark(stage, onGone);
    }
  });

  useEffect(() => {
    const stage = stageRef.current;
    if (stage === null) {
      throw new Error("the startup layer mounted without its stage");
    }
    writePref("startupPlayed", true);
    const playing = new AbortController();
    void play({
      stage,
      signal: playing.signal,
      stopAt,
      end: (reduced) => {
        end(stage, reduced);
      },
    });
    return () => {
      playing.abort();
    };
  }, []);

  return (
    <div
      ref={stageRef}
      // oxlint-disable-next-line jsx-a11y/prefer-tag-over-role -- the sequence is drawn in layers inside this stage
      role="img"
      aria-label="SlipStream"
      className={`startup-stage z-100 transition-opacity duration-360 ease-standard ${fading ? "pointer-events-none opacity-0" : ""}`}
      onTransitionEnd={onGone}
    />
  );
}
