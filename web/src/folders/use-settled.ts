import { useEffect, useState } from "react";

/** The value once it has held still for `ms`, starting from `initial`. */
export function useSettled<T>(value: T, ms: number, initial: T = value): T {
  const [settled, setSettled] = useState(initial);
  useEffect(() => {
    const timer = setTimeout(() => {
      setSettled(value);
    }, ms);
    return () => {
      clearTimeout(timer);
    };
  }, [value, ms]);
  return settled;
}
