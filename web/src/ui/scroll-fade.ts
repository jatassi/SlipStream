import { useCallback, useState } from "react";

type Fade = "start" | "end" | "both";

/** Fades a sideways scroller's edge where more of it lies past that edge, so a clipped option reads as one to scroll to. */
export const scrollFadeLook =
  "data-[fade=end]:[mask-image:linear-gradient(to_right,#000_calc(100%-2rem),transparent)] data-[fade=start]:[mask-image:linear-gradient(to_left,#000_calc(100%-2rem),transparent)] data-[fade=both]:[mask-image:linear-gradient(to_right,transparent,#000_2rem,#000_calc(100%-2rem),transparent)]";

function fadeOf(node: HTMLElement): Fade | undefined {
  const start = node.scrollLeft > 1;
  const end = node.scrollLeft + node.clientWidth < node.scrollWidth - 1;
  if (start && end) {
    return "both";
  }
  if (start) {
    return "start";
  }
  return end ? "end" : undefined;
}

function follow(node: HTMLElement, onFade: (fade: Fade | undefined) => void): () => void {
  const measure = () => {
    onFade(fadeOf(node));
  };
  measure();
  const observer = new ResizeObserver(measure);
  observer.observe(node);
  node.addEventListener("scroll", measure, { passive: true });
  return () => {
    observer.disconnect();
    node.removeEventListener("scroll", measure);
  };
}

/** The ref for a sideways scroller and the edge it should fade at, measured as it scrolls and resizes. */
export function useScrollFade() {
  const [fade, setFade] = useState<Fade>();
  const ref = useCallback(
    (node: HTMLElement | null) => (node === null ? undefined : follow(node, setFade)),
    [],
  );
  return { ref, fade };
}
