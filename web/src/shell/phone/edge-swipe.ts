import { useRouter } from "@tanstack/react-router";
import { useEffect } from "react";

import { useMedia } from "../media";

const edge = 20;
const travel = 80;

// An installed iOS web app has no swipe-back of its own, so a drag that starts
// at the left edge goes back.
export function useEdgeSwipeBack(): void {
  const router = useRouter();
  const standalone = useMedia("(display-mode: standalone)");
  useEffect(() => {
    const listening = new AbortController();
    if (standalone) {
      follow(listening.signal, () => {
        router.history.back();
      });
    }
    return () => {
      listening.abort();
    };
  }, [router, standalone]);
}

function follow(signal: AbortSignal, back: () => void) {
  let start: { x: number; y: number } | undefined;
  window.addEventListener(
    "pointerdown",
    (event) => {
      start = event.clientX <= edge ? { x: event.clientX, y: event.clientY } : undefined;
    },
    { signal },
  );
  window.addEventListener(
    "pointerup",
    (event) => {
      if (start === undefined) {
        return;
      }
      const dx = event.clientX - start.x;
      const dy = Math.abs(event.clientY - start.y);
      start = undefined;
      if (dx >= travel && dx > dy) {
        back();
      }
    },
    { signal },
  );
  window.addEventListener(
    "pointercancel",
    () => {
      start = undefined;
    },
    { signal },
  );
}
