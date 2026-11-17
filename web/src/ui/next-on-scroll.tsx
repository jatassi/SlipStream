import { useEffect, useEffectEvent, useRef } from "react";

/** Loads the next page once the end of a phone's list scrolls into view. */
export function NextOnScroll({ onNext }: { onNext: () => void }) {
  const end = useRef<HTMLDivElement>(null);
  const next = useEffectEvent(onNext);
  useEffect(() => {
    const watcher = new IntersectionObserver((entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        next();
      }
    });
    if (end.current !== null) {
      watcher.observe(end.current);
    }
    return () => {
      watcher.disconnect();
    };
  }, []);
  return <div ref={end} aria-hidden="true" className="h-px" />;
}
