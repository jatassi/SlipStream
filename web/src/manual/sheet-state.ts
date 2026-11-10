import { useSyncExternalStore } from "react";

import { sentenceOf } from "../api/errors";
import type { GrabsStartManual } from "../api/gen/model";
import { startManualSearch } from "../api/gen/query/grabs/grabs";
import { createStore } from "../shell/store";

/** What a Manual search covers: the Work, some of its levels or Items, or one cell. */
export type ManualScope = Pick<GrabsStartManual, "levelIds" | "itemIds" | "cell">;

export type Start =
  | { readonly state: "starting" }
  | { readonly state: "started" }
  | { readonly state: "refused"; readonly sentence: string };

export type ManualOpening = {
  /** Tells one opening from the next, so a late answer to an earlier start is dropped. */
  readonly serial: number;
  readonly workId: string;
  readonly scope: ManualScope;
  readonly start: Start;
};

/** A closing sheet keeps its opening through the exit animation. */
type SheetState =
  | { readonly shown: "none" }
  | { readonly shown: "open" | "closing"; readonly opening: ManualOpening };

type SheetEvent =
  | { readonly kind: "open"; readonly opening: ManualOpening }
  | { readonly kind: "settle"; readonly serial: number; readonly start: Start }
  | { readonly kind: "close" };

const sheet = createStore<SheetState, SheetEvent>(
  (state, event) => {
    if (event.kind === "open") {
      return { shown: "open", opening: event.opening };
    }
    if (state.shown === "none") {
      return state;
    }
    if (event.kind === "close") {
      return state.shown === "open" ? { shown: "closing", opening: state.opening } : state;
    }
    return state.opening.serial === event.serial
      ? { ...state, opening: { ...state.opening, start: event.start } }
      : state;
  },
  { shown: "none" },
);

let serials = 0;

async function start(serial: number, workId: string, scope: ManualScope) {
  let started: Start = { state: "started" };
  try {
    await startManualSearch(workId, scope);
  } catch (error) {
    started = { state: "refused", sentence: sentenceOf(error) };
  }
  sheet.dispatch({ kind: "settle", serial, start: started });
}

/**
 * Opens the Manual search sheet on a Work and starts its Manual search for
 * the scope. A Work whose Manual search already has this scope keeps it, so
 * reopening shows the last one with its age.
 */
export function openManualSearch({ workId, scope = {} }: { workId: string; scope?: ManualScope }) {
  serials += 1;
  const serial = serials;
  sheet.dispatch({
    kind: "open",
    opening: { serial, workId, scope, start: { state: "starting" } },
  });
  void start(serial, workId, scope);
}

export const closeManualSearch = () => {
  sheet.dispatch({ kind: "close" });
};

export const useManualSheet = (): SheetState =>
  useSyncExternalStore(sheet.subscribe, sheet.getSnapshot);
