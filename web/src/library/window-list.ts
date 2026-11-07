import { useLayoutEffect, useRef, useState } from "react";

/** Where an element starts on the page, which a window virtualizer measures scroll from. */
export function useScrollMargin<E extends HTMLElement>() {
  const ref = useRef<E>(null);
  const [margin, setMargin] = useState(0);
  useLayoutEffect(() => {
    const measure = () => {
      setMargin((ref.current?.getBoundingClientRect().top ?? 0) + window.scrollY);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(document.body);
    return () => {
      observer.disconnect();
    };
  }, []);
  return { ref, margin };
}

/** An element's width, kept current as its container reflows. */
export function useWidth<E extends HTMLElement>() {
  const ref = useRef<E>(null);
  const [width, setWidth] = useState(0);
  useLayoutEffect(() => {
    const observer = new ResizeObserver(([entry]) => {
      setWidth(entry?.contentRect.width ?? 0);
    });
    if (ref.current !== null) {
      observer.observe(ref.current);
    }
    return () => {
      observer.disconnect();
    };
  }, []);
  return { ref, width };
}
