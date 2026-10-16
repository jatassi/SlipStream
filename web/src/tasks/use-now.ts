import { useEffect, useState } from "react";

import { type Clock, knownZone } from "../format";

export function useNow(everyMs: number): Date {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const timer = setInterval(() => {
      setNow(new Date());
    }, everyMs);
    return () => {
      clearInterval(timer);
    };
  }, [everyMs]);
  return now;
}

/**
 * Now on the Controller's clock: its zone when the browser knows the name,
 * else the browser's own.
 */
export function useClock(everyMs: number, zone: string): Clock {
  return { now: useNow(everyMs), zone: knownZone(zone) };
}
