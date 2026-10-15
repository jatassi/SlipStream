import { useSyncExternalStore } from "react";

export type Flag = {
  readonly set: (value: boolean) => void;
  readonly get: () => boolean;
  readonly useValue: () => boolean;
};

export function createFlag(): Flag {
  let value = false;
  const listeners = new Set<() => void>();
  const subscribe = (listener: () => void) => {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  };
  const get = () => value;
  return {
    set: (next) => {
      value = next;
      for (const listener of listeners) {
        listener();
      }
    },
    get,
    useValue: () => useSyncExternalStore(subscribe, get),
  };
}
