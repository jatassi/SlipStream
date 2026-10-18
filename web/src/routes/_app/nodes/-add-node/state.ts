import type { NodesCodeState, NodesJoinCode, NodesNode } from "../../../../api/gen/model";
import { reachedBy } from "../../../../nodes/dial";

/** What getJoinCode answered for the code the dialog follows. */
export type CodeAnswer =
  | { readonly kind: "loading" }
  | { readonly kind: "gone" }
  | { readonly kind: "failed"; readonly sentence: string }
  | { readonly kind: "known"; readonly state: NodesCodeState };

/** The Controller's dial to a pasted Node address. */
export type Dial =
  | { readonly kind: "none" }
  | { readonly kind: "dialing"; readonly addresses: readonly string[] }
  | { readonly kind: "cant-reach"; readonly addresses: readonly string[] };

export type AddNodeState =
  | { readonly kind: "loading" }
  | { readonly kind: "failed"; readonly sentence: string }
  | {
      readonly kind: "open";
      /** Absent once the page reloads: the Controller keeps only the code's hash. */
      readonly code: string | undefined;
      readonly left: string | undefined;
      readonly dial: Dial;
    }
  | { readonly kind: "expired" }
  | { readonly kind: "revoked" }
  | { readonly kind: "pairing" }
  | { readonly kind: "paired"; readonly node: NodesNode; readonly reached: string };

export type Facts = {
  readonly id: string;
  readonly answer: CodeAnswer;
  /** The code this page made, the one time its text is known. */
  readonly minted: NodesJoinCode | undefined;
  /** From minted, or from the open codes once the page reloads. */
  readonly expiresAt: string | undefined;
  readonly nodes: readonly NodesNode[];
  readonly now: Date;
};

const second = 1000;
const minute = 60 * second;
const hour = 60 * minute;
const day = 24 * hour;

const two = (n: number) => String(n).padStart(2, "0");

/** The time left on a code, ticking: "59:42", "5:59:42", or "2 days 5 h" past a day. */
export function countdown(ms: number): string {
  const days = Math.floor(ms / day);
  const hours = Math.floor((ms % day) / hour);
  if (days > 0) {
    return `${days} ${days === 1 ? "day" : "days"} ${hours} h`;
  }
  const minutes = Math.floor((ms % hour) / minute);
  const seconds = Math.floor((ms % minute) / second);
  return hours > 0 ? `${hours}:${two(minutes)}:${two(seconds)}` : `${minutes}:${two(seconds)}`;
}

function dialOf(state: NodesCodeState): Dial {
  const { dialing } = state;
  if (dialing === undefined) {
    return { kind: "none" };
  }
  return { kind: dialing.unreachable ? "cant-reach" : "dialing", addresses: dialing.addresses };
}

function usedState(state: NodesCodeState, nodes: readonly NodesNode[]): AddNodeState {
  const node = nodes.find(({ id }) => id === state.nodeId);
  const controller = nodes.find(({ self }) => self);
  if (node === undefined || controller === undefined) {
    return { kind: "pairing" };
  }
  return { kind: "paired", node, reached: reachedBy(controller.name, node) };
}

/** Where the Add a Node dialog stands, from the server's answers and the clock alone. */
export function addNodeState({ id, answer, minted, expiresAt, nodes, now }: Facts): AddNodeState {
  const left = expiresAt === undefined ? undefined : Date.parse(expiresAt) - now.getTime();
  switch (answer.kind) {
    case "loading":
    case "failed":
      return answer;
    case "gone":
      return left !== undefined && left <= 0 ? { kind: "expired" } : { kind: "revoked" };
    case "known":
      break;
  }
  const { state } = answer;
  if (state.state === "used") {
    return usedState(state, nodes);
  }
  if (state.state === "expired" || (left !== undefined && left <= 0)) {
    return { kind: "expired" };
  }
  return {
    kind: "open",
    code: minted?.id === id ? minted.code : undefined,
    left: left === undefined ? undefined : countdown(left),
    dial: dialOf(state),
  };
}
