import type {
  GrabsClientChoice,
  GrabsGrabPick,
  GrabsLiftKey,
  GrabsLiftKeyCode,
  GrabsPickResult,
  GrabsPinned,
  GrabsPlanPick,
  GrabsPlanRow,
  GrabsWordedReason,
} from "../api/gen/model";
import type { ClientPick } from "../grabs/send-to-sheet";

/** A client by its ClientPick key, or routed for where Tracker routing sends it. */
export const routed = "routed";

/**
 * What the Admin chose on one Manual search's plan, by row key. A row is
 * ticked unless unticked, so a row that streams in later is ticked, and an
 * untick sticks while its key does.
 */
export type Choices = {
  readonly unticked: ReadonlySet<string>;
  /** Each row's own client; a row without one goes where Send all to sends. */
  readonly clients: ReadonlyMap<string, string>;
  readonly sendAll: string;
};

/** The last grab's result for each row key, and for each Release grabbed from All Releases. */
export type Results = ReadonlyMap<string, GrabsPickResult>;

export const noChoices: Choices = {
  unticked: new Set(),
  clients: new Map(),
  sendAll: routed,
};

/** The pins with the row's pin replaced by this one. */
export const swap = (pins: readonly GrabsPinned[], key: string, pin: string): GrabsPinned[] => [
  ...pins.filter((pinned) => pinned.key !== key),
  { key, pin },
];

export const clientOf = (choices: Choices, key: string): string =>
  choices.clients.get(key) ?? choices.sendAll;

export const grabbed = (results: Results, key: string): boolean =>
  (results.get(key)?.grabId ?? null) !== null;

export type PickRow = GrabsPlanRow & { readonly pick: GrabsPlanPick };

export const hasPick = (row: GrabsPlanRow): row is PickRow => row.pick !== null;

/** A row the Grab button sends: ticked, with a pick, and not grabbed already. */
export const sends = (choices: Choices, results: Results, row: GrabsPlanRow): row is PickRow =>
  hasPick(row) && !choices.unticked.has(row.key) && !grabbed(results, row.key);

const liftable: Record<GrabsLiftKeyCode, true> = {
  wrong_title: true,
  size_below: true,
  size_above: true,
  score_below_minimum: true,
  repack_other_group: true,
};

const isLift = (code: string): code is GrabsLiftKeyCode => Object.hasOwn(liftable, code);

/** The lifts the server worded, sent back as codes and Targets; it words only soft refusals as lifts. */
const liftsOf = (lifts: readonly GrabsWordedReason[]): GrabsLiftKey[] =>
  lifts.flatMap(({ code, targetId }) =>
    isLift(code) ? [{ code, ...(targetId !== undefined && { targetId }) }] : [],
  );

/** One pick as the server sent it, with the sheet's own client and a Grab to cancel first. */
export function pickOf(
  offer: {
    readonly release: { readonly id: string; readonly blocked: string | null };
    readonly cells: readonly string[];
    readonly lifts: readonly GrabsWordedReason[];
    readonly edition: string | null;
  },
  client: GrabsClientChoice | undefined,
  cancelGrabId?: string,
): GrabsGrabPick {
  return {
    releaseId: offer.release.id,
    cells: [...offer.cells],
    lifts: liftsOf(offer.lifts),
    ...(offer.edition !== null && { edition: offer.edition }),
    ...(client !== undefined && { client }),
    ...(offer.release.blocked !== null && { unblock: true }),
    ...(cancelGrabId !== undefined && { cancelGrabId }),
  };
}

export const bodyOf = (picks: readonly ClientPick[], key: string): GrabsClientChoice | undefined =>
  picks.find((pick) => pick.key === key)?.body;
