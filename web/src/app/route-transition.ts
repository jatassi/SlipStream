import type { HistoryAction, RouterHistory } from "@tanstack/history";

export type TransitionType = "push" | "pop";

const types: { readonly [A in HistoryAction]: TransitionType | null } = {
  PUSH: "push",
  FORWARD: "push",
  BACK: "pop",
  REPLACE: null,
  GO: null,
};

export const transitionTypeFor = (
  action: HistoryAction,
  popstate?: PopStateEvent,
): TransitionType | null => (popstate?.hasUAVisualTransition === true ? null : types[action]);

export function routeTransitions(history: RouterHistory) {
  let popstate: PopStateEvent | undefined;
  let type: TransitionType | null = null;
  const onPopState = (event: PopStateEvent) => {
    popstate = event;
  };
  window.addEventListener("popstate", onPopState, { capture: true });
  const unsubscribe = history.subscribe(({ action }) => {
    const traversal = action.type === "BACK" || action.type === "FORWARD";
    type = transitionTypeFor(action.type, traversal ? popstate : undefined);
    popstate = undefined;
  });
  return {
    types: (): TransitionType[] | false => {
      const next = type;
      type = null;
      return next === null ? false : [next];
    },
    dispose: () => {
      window.removeEventListener("popstate", onPopState, { capture: true });
      unsubscribe();
    },
  };
}
