export type Store<S, E> = {
  readonly getSnapshot: () => S;
  readonly subscribe: (listener: () => void) => () => void;
  readonly dispatch: (event: E) => void;
};

/** A client-side store over a pure reducer, for `useSyncExternalStore`. */
export function createStore<S, E>(reducer: (state: S, event: E) => S, initial: S): Store<S, E> {
  let state = initial;
  const listeners = new Set<() => void>();
  return {
    getSnapshot: () => state,
    subscribe: (listener) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    dispatch: (event) => {
      const next = reducer(state, event);
      if (next === state) {
        return;
      }
      state = next;
      for (const listener of listeners) {
        listener();
      }
    },
  };
}
