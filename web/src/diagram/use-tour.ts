import { useEffect, useRef, useState } from "react";

import { type Part, stepPart } from "./parts-order";

const TOUR_EVERY = 4000;
const IDLE_BEFORE_TOUR = 4000;

/**
 * The idle tour: while nothing is touched it previews the next part every
 * 4 s. Any input holds it for 4 s. It returns the hold.
 */
export function useTour(
  paused: boolean,
  parts: readonly Part[],
  setPreview: (step: (current: Part | undefined) => Part | undefined) => void,
): () => void {
  const [touring, setTouring] = useState(true);
  const idle = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    const timer = setInterval(() => {
      if (touring && !paused) {
        setPreview((current) => stepPart(parts, current, 1));
      }
    }, TOUR_EVERY);
    return () => {
      clearInterval(timer);
    };
  }, [touring, paused, parts, setPreview]);
  useEffect(
    () => () => {
      if (idle.current !== null) {
        clearTimeout(idle.current);
      }
    },
    [],
  );
  return () => {
    setTouring(false);
    if (idle.current !== null) {
      clearTimeout(idle.current);
    }
    idle.current = setTimeout(() => {
      setTouring(true);
    }, IDLE_BEFORE_TOUR);
  };
}
