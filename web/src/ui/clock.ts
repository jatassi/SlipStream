import { useSyncExternalStore } from "react";

/** Now, read on a clock in a time zone; undefined is the person's own. */
export type Clock = { readonly now: Date; readonly zone: string | undefined };

/** How often a reader of the shared clock re-renders. */
export type Tick = "minute" | "second";

const ticks = ["minute", "second"] as const satisfies readonly Tick[];

const unitOf: { readonly [T in Tick]: number } = { minute: 60_000, second: 1000 };

const readers: { readonly [T in Tick]: Set<() => void> } = {
  minute: new Set(),
  second: new Set(),
};

const shown: { [T in Tick]: Date } = { minute: new Date(), second: new Date() };

let timer: ReturnType<typeof setTimeout> | undefined;

const unitAt = (time: number, tick: Tick) => Math.floor(time / unitOf[tick]);

function publish(now: Date) {
  for (const tick of ticks) {
    if (unitAt(now.getTime(), tick) !== unitAt(shown[tick].getTime(), tick)) {
      shown[tick] = now;
      for (const reader of readers[tick]) {
        reader();
      }
    }
  }
}

function finest(): Tick | undefined {
  if (readers.second.size > 0) {
    return "second";
  }
  return readers.minute.size > 0 ? "minute" : undefined;
}

function arm() {
  clearTimeout(timer);
  timer = undefined;
  const tick = finest();
  if (tick === undefined || document.visibilityState === "hidden") {
    return;
  }
  const unit = unitOf[tick];
  timer = setTimeout(
    () => {
      publish(new Date());
      arm();
    },
    unit - (Date.now() % unit),
  );
}

function onVisibility() {
  if (document.visibilityState === "visible") {
    publish(new Date());
  }
  arm();
}

const subscribers = () => readers.minute.size + readers.second.size;

const subscribeTo = (tick: Tick) => (reader: () => void) => {
  if (subscribers() === 0) {
    document.addEventListener("visibilitychange", onVisibility);
  }
  // The shown now has stood still since the last reader left, so the first
  // reader of a tick takes the current one, to the millisecond.
  if (readers[tick].size === 0) {
    shown[tick] = new Date();
  }
  readers[tick].add(reader);
  arm();
  return () => {
    readers[tick].delete(reader);
    if (subscribers() === 0) {
      document.removeEventListener("visibilitychange", onVisibility);
    }
    arm();
  };
};

const subscriptions: { readonly [T in Tick]: (reader: () => void) => () => void } = {
  minute: subscribeTo("minute"),
  second: subscribeTo("second"),
};

// With no reader, nothing has moved the shown now along, so the first render
// after a quiet spell (or under a test's faked Date) reads the current one.
function snapshotOf(tick: Tick): Date {
  if (
    readers[tick].size === 0 &&
    unitAt(Date.now(), tick) !== unitAt(shown[tick].getTime(), tick)
  ) {
    shown[tick] = new Date();
  }
  return shown[tick];
}

const snapshots: { readonly [T in Tick]: () => Date } = {
  minute: () => snapshotOf("minute"),
  second: () => snapshotOf("second"),
};

/**
 * Now from the one shared clock, which every reader shares: "minute" turns on
 * each wall minute, "second" each second while it is read. No timer runs while
 * the tab is hidden, and turning visible reads now at once.
 */
export function useNow(tick: Tick = "minute"): Date {
  return useSyncExternalStore(subscriptions[tick], snapshots[tick]);
}

/** The person's own time zone, the browser's. */
export const ownZone = (): string => Intl.DateTimeFormat().resolvedOptions().timeZone;

/**
 * The zone when the name is an IANA one Intl knows, else undefined for the
 * person's own. Intl reads some abbreviations as an unrelated zone (BST as
 * Asia/Dhaka), so only an Area/Location name or UTC counts.
 */
export function knownZone(name: string): string | undefined {
  if (name !== "UTC" && !name.includes("/")) {
    return undefined;
  }
  try {
    return new Intl.DateTimeFormat("en", { timeZone: name }).resolvedOptions().timeZone;
  } catch {
    return undefined;
  }
}

/** Now on the person's clock, or on a zone's when a time answers to a setting stated in it. */
export function useClock(zone?: string): Clock {
  return { now: useNow(), zone: zone === undefined ? undefined : knownZone(zone) };
}
