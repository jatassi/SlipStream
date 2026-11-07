import { useSyncExternalStore } from "react";

import type { WorksCandidate, WorksContentType } from "../../api/gen/model";
import { createStore } from "../../shell/store";

export type Adding = { readonly type: WorksContentType; readonly candidate: WorksCandidate };

/** A closing sheet keeps its Work through the exit animation. */
type SheetState =
  | { readonly shown: "none" }
  | { readonly shown: "open"; readonly adding: Adding }
  | { readonly shown: "closing"; readonly adding: Adding };

type SheetEvent = { readonly kind: "open"; readonly adding: Adding } | { readonly kind: "close" };

const sheet = createStore<SheetState, SheetEvent>(
  (state, event) => {
    if (event.kind === "open") {
      return { shown: "open", adding: event.adding };
    }
    return state.shown === "open" ? { shown: "closing", adding: state.adding } : state;
  },
  { shown: "none" },
);

export const openAddSheet = (adding: Adding) => {
  sheet.dispatch({ kind: "open", adding });
};

export const closeAddSheet = () => {
  sheet.dispatch({ kind: "close" });
};

export const useAddSheet = (): SheetState =>
  useSyncExternalStore(sheet.subscribe, sheet.getSnapshot);
