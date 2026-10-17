import { useSyncExternalStore } from "react";

type Ask = {
  readonly id: number;
  readonly confirmed: Promise<boolean>;
  readonly settle: (confirmed: boolean) => void;
};

let asks = 0;
let current: Ask | undefined;
const listeners = new Set<() => void>();

function notify() {
  for (const listener of listeners) {
    listener();
  }
}

/**
 * Asks the person to confirm it's them. Every caller that asks while the
 * question is open shares it, and each learns whether they confirmed.
 */
export function askReconfirm(): Promise<boolean> {
  if (current === undefined) {
    asks += 1;
    let resolve!: (confirmed: boolean) => void;
    const confirmed = new Promise<boolean>((done) => {
      resolve = done;
    });
    current = {
      id: asks,
      confirmed,
      settle: (answer) => {
        current = undefined;
        notify();
        resolve(answer);
      },
    };
    notify();
  }
  return current.confirmed;
}

export function settleReconfirm(confirmed: boolean): void {
  current?.settle(confirmed);
}

const subscribe = (listener: () => void) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};

export const useReconfirmAsk = (): number | undefined =>
  useSyncExternalStore(subscribe, () => current?.id);
