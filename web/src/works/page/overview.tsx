import { useLayoutEffect, useRef, useState } from "react";

// Three lines, with More while the text runs past them.
export function Overview({ text }: { text: string }) {
  const ref = useRef<HTMLParagraphElement>(null);
  const [open, setOpen] = useState(false);
  const [clipped, setClipped] = useState(false);
  useLayoutEffect(() => {
    const element = ref.current;
    const measure = () => {
      setClipped(element !== null && element.scrollHeight > element.clientHeight + 1);
    };
    measure();
    const observer = new ResizeObserver(measure);
    if (element !== null) {
      observer.observe(element);
    }
    return () => {
      observer.disconnect();
    };
  }, []);
  return (
    <div className="flex max-w-[78ch] flex-col items-start gap-1">
      <p ref={ref} className={`text-ink-2 ${open ? "" : "line-clamp-3"}`}>
        {text}
      </p>
      {(clipped || open) && (
        <button
          type="button"
          className="font-medium text-ink-1 hover:underline"
          onClick={() => {
            setOpen(!open);
          }}
        >
          {open ? "Less" : "More"}
        </button>
      )}
    </div>
  );
}
