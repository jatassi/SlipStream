import { type FocusEvent, type PointerEvent, useEffect, useMemo, useRef, useState } from "react";

import type { HomeJob } from "../api/gen/model";
import { type AmbientSource, useAmbient } from "../shell/ambient/use-ambient";
import { sized } from "../ui/artwork";
import { useShell } from "../ui/shell-context";

// DESIGN.md's Admin Home: a pointed row tints after a short pause so skimming
// doesn't flash colours, and the tint returns a moment after leaving.
const enterDelay = 90;
const returnDelay = 450;

const rowPrefix = "job-";

export const jobAnchor = (job: HomeJob): string => `${rowPrefix}${job.grab.id}`;

const neutral: AmbientSource = { kind: "neutral" };

const grabOf = (target: EventTarget | null): string | undefined => {
  const row = target instanceof Element ? target.closest(`[id^="${rowPrefix}"]`) : null;
  return row?.id.slice(rowPrefix.length);
};

/** The Grabs whose Jobs rows cross the viewport's middle band, watched on phone only. */
function useBanded(jobs: readonly HomeJob[], on: boolean): ReadonlySet<string> {
  const [inBand, setInBand] = useState<ReadonlySet<string>>(new Set());
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        setInBand((prior) => {
          const next = new Set(prior);
          for (const { target, isIntersecting } of entries) {
            const id = target.id.slice(rowPrefix.length);
            if (isIntersecting) {
              next.add(id);
            } else {
              next.delete(id);
            }
          }
          return next;
        });
      },
      { rootMargin: "-45% 0px -45% 0px" },
    );
    for (const job of on ? jobs : []) {
      const row = document.querySelector(`#${jobAnchor(job)}`);
      if (row !== null) {
        observer.observe(row);
      }
    }
    return () => {
      observer.disconnect();
    };
  }, [jobs, on]);
  return inBand;
}

/** The Grab whose row the pointer or focus rests on, after the enter delay, and none after the return delay. */
function usePointed() {
  const [pointed, setPointed] = useState<string>();
  const aimed = useRef<string>(undefined);
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(
    () => () => {
      clearTimeout(timer.current);
    },
    [],
  );
  const aim = (id: string | undefined) => {
    if (aimed.current === id) {
      return;
    }
    aimed.current = id;
    clearTimeout(timer.current);
    timer.current = setTimeout(
      () => {
        setPointed(id);
      },
      id === undefined ? returnDelay : enterDelay,
    );
  };
  const handlers = {
    onPointerOver: (event: PointerEvent) => {
      aim(grabOf(event.target));
    },
    onPointerLeave: () => {
      aim(undefined);
    },
    onFocus: (event: FocusEvent) => {
      aim(grabOf(event.target));
    },
    onBlur: (event: FocusEvent) => {
      if (grabOf(event.relatedTarget) === undefined) {
        aim(undefined);
      }
    },
  };
  return { pointed, handlers };
}

/**
 * Home's one ambient source. In a desktop browser the row under the pointer
 * or focus, on phone the topmost row in the middle band, else the top Job
 * whose Work hasn't failed. A Work with no backdrop, or one that has failed,
 * leaves it neutral. Returns the Jobs frame's handlers.
 */
export function useHomeTint(jobs: readonly HomeJob[]) {
  const desktop = useShell() === "desktop";
  const { pointed, handlers } = usePointed();
  const inBand = useBanded(jobs, !desktop);
  const chosen =
    jobs.find(({ grab }) => (desktop ? grab.id === pointed : inBand.has(grab.id))) ??
    jobs.find(({ work }) => !work.failed);
  const workId = chosen?.work.id;
  const backdrop = chosen?.work.backdropUrl ?? null;
  const failed = chosen?.work.failed ?? true;
  const source = useMemo<AmbientSource>(
    () =>
      workId === undefined || failed || backdrop === null
        ? neutral
        : { kind: "artwork", url: sized(backdrop, "large"), workId },
    [workId, backdrop, failed],
  );
  useAmbient(source, { returnDelay: desktop ? returnDelay : 0 });
  return desktop ? handlers : {};
}
