import { z } from "zod";

import { isApiError } from "../api/errors";
import type {
  NodesAssignment,
  NodesBoardNode,
  NodesHeld,
  NodesNode,
  NodesOutcome,
  NodesRoleBoard,
  NodesRoleEntry,
  NodesRoleGroup,
  NodesRoleView,
  NodesRoleViewState,
} from "../api/gen/model";
import { NodesOutcome as OutcomeSchema } from "../api/gen/zod/slipStream.schemas.zod";
import { type Clock, bytes } from "../format";
import { awayLine } from "./standing";

export type RoleName = NodesRoleEntry["role"];

/** The Roles an Admin has ticked on one Node, before they review the change. */
export type Draft = ReadonlySet<RoleName>;

export function seatOf(board: NodesRoleBoard, nodeId: string): NodesBoardNode {
  const seat = board.nodes.find((candidate) => candidate.id === nodeId);
  if (seat === undefined) {
    throw new Error(`the role board has no Node ${nodeId}`);
  }
  return seat;
}

export function labelOf(board: NodesRoleBoard, role: RoleName): string {
  const entry = board.groups
    .flatMap(({ roles }) => roles)
    .find((candidate) => candidate.role === role);
  if (entry === undefined) {
    throw new Error(`the role board has no Role ${role}`);
  }
  return entry.label;
}

/** The Roles the Controller holds whatever is ticked, so its section ticks them for good. */
export const lockedOf = (board: NodesRoleBoard, seat: NodesBoardNode): Draft =>
  new Set(
    seat.controller
      ? board.groups.flatMap(({ roles }) =>
          roles.filter(({ pinned }) => pinned).map(({ role }) => role),
        )
      : [],
  );

export const draftOf = (board: NodesRoleBoard, seat: NodesBoardNode): Draft =>
  new Set([...seat.assigned, ...lockedOf(board, seat)]);

export const sameDraft = (a: Draft, b: Draft): boolean =>
  a.size === b.size && [...a].every((role) => b.has(role));

export function withRole(draft: Draft, role: RoleName, on: boolean): Draft {
  const next = new Set(draft);
  if (on) {
    next.add(role);
  } else {
    next.delete(role);
  }
  return next;
}

/** Ticks or unticks every Role of group, leaving the locked ones as they are. */
export const withGroup = (draft: Draft, group: NodesRoleGroup, on: boolean, locked: Draft): Draft =>
  group.roles
    .filter(({ role }) => !locked.has(role))
    .reduce((next, { role }) => withRole(next, role, on), draft);

export const assignmentOf = (nodeId: string, draft: Draft): NodesAssignment => ({
  nodeId,
  roles: [...draft],
});

/** "3 torrents (41 GB)", as the Review writes held work. */
export const heldWords = ({ units, noun, bytes: size }: NodesHeld): string =>
  size > 0 ? `${units} ${noun} (${bytes(size)})` : `${units} ${noun}`;

const listed = (names: readonly string[]): string =>
  names.length < 2 ? names.join("") : `${names.slice(0, -1).join(", ")} and ${names.at(-1)}`;

const sentence = (words: string): string => words.charAt(0).toUpperCase() + words.slice(1);

type Standing = Pick<NodesNode, "standing"> & { readonly since: string };

const awayWords = ({ standing, since }: Standing, clock: Clock): string =>
  standing === "in-step" ? "Reconnecting" : sentence(awayLine(standing, since, clock));

const lines: {
  readonly [S in NodesRoleViewState]: (
    view: NodesRoleView,
    standing: Standing,
    covered: readonly string[],
    clock: Clock,
  ) => string;
} = {
  holding: ({ held }) => (held === undefined ? "Holding" : `Holding ${heldWords(held)}`),
  away: (_view, standing, _covered, clock) => awayWords(standing, clock),
  lapsed: (_view, standing, _covered, clock) =>
    `${awayWords(standing, clock)} · past the offline grace`,
  draining: ({ held }) => (held === undefined ? "Draining" : `Draining · ${heldWords(held)} left`),
  "writing-off": ({ held }) =>
    held === undefined ? "Writing off" : `Writing off ${heldWords(held)}`,
  covering: (_view, _standing, covered) =>
    covered.length === 0 ? "Covering" : `Covering for ${listed(covered)}`,
};

/** The Nodes whose Role has lapsed, which the Controller covers for. */
export const coveredFor = (board: NodesRoleBoard, role: RoleName): string[] =>
  board.nodes
    .filter(({ roles }) => roles.some((view) => view.role === role && view.state === "lapsed"))
    .map(({ name }) => name);

/** What a Role is doing on its Node: "Draining · 3 torrents left", "Offline since 21:31". */
export const roleLine = (
  board: NodesRoleBoard,
  view: NodesRoleView,
  standing: Standing,
  clock: Clock,
): string => lines[view.state](view, standing, coveredFor(board, view.role), clock);

export type Hint = { readonly tone: "warn" | "block"; readonly text: string };

/**
 * Why a Role may not do well on a Node: its probe found the Node weak or
 * failing, or, while the Role isn't ticked there, a guardrail warns about or
 * blocks adding it.
 */
export function hintOf(
  board: NodesRoleBoard,
  seat: NodesBoardNode,
  entry: NodesRoleEntry,
): Hint | undefined {
  const probe = seat.probes.find(({ probe: name }) => name === entry.probe);
  const found = probe?.fit === "fits" ? undefined : probe?.words;
  const add = seat.assigned.includes(entry.role)
    ? undefined
    : board.gestures.find(
        ({ role, from, to }) => role === entry.role && from === undefined && to === seat.id,
      );
  if (add !== undefined && add.verdict !== "allow") {
    return { tone: add.verdict, text: found === undefined ? add.word : `${add.word}. ${found}` };
  }
  if (found !== undefined) {
    return { tone: probe?.fit === "fails" ? "block" : "warn", text: found };
  }
  return undefined;
}

const Carrying = z.object({ outcome: OutcomeSchema });

/**
 * The Outcome a refusal carries: a plan gone stale since it was read, a
 * guardrail that blocks it, or a write-off with nothing to give up.
 */
export function refusedOutcome(error: unknown): NodesOutcome | undefined {
  if (!isApiError(error)) {
    return undefined;
  }
  const carried = Carrying.safeParse(error.body);
  return carried.success ? carried.data.outcome : undefined;
}

export const blocks = (outcome: NodesOutcome): string[] =>
  outcome.checks.filter(({ verdict }) => verdict === "block").map(({ words }) => words);
