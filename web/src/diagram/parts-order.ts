import type { Landing } from "../routes/_app/nodes/-landing/view";

/** A part of the Diagram that can be selected: a Node's tile, one of its Roles, or a link by its Worker. */
export type Part = `node:${string}` | `role:${string}:${string}` | `link:${string}`;

/** Whether a data-part value names a part. */
export const isPart = (value: string): value is Part =>
  /^(?:node|link):[^:]+$|^role:[^:]+:[^:]+$/u.test(value);

/** Every part in the order the arrow keys walk them: each Node, then its Roles; then the links. */
export const partsOf = (landing: Landing): Part[] => [
  ...landing.nodes.flatMap((node): Part[] => [
    `node:${node.id}`,
    ...node.roles.map(({ role }): Part => `role:${node.id}:${role}`),
  ]),
  ...landing.links.map(({ worker }): Part => `link:${worker}`),
];

/** The part `by` steps from `from`, wrapping round; the first part when nothing is selected. */
export function stepPart(
  parts: readonly Part[],
  from: Part | undefined,
  by: number,
): Part | undefined {
  if (parts.length === 0) {
    return undefined;
  }
  const at = from === undefined ? -1 : parts.indexOf(from);
  const next = at < 0 ? (by > 0 ? 0 : parts.length - 1) : (at + by + parts.length) % parts.length;
  return parts[next];
}

export type Selected =
  | { readonly kind: "node"; readonly node: string }
  | { readonly kind: "role"; readonly node: string; readonly role: string }
  | { readonly kind: "link"; readonly worker: string };

export function selectedOf(part: Part): Selected {
  const [kind, node = "", role = ""] = part.split(":");
  if (kind === "node") {
    return { kind, node };
  }
  if (kind === "role") {
    return { kind, node, role };
  }
  return { kind: "link", worker: node };
}

/** A part in words, for the screen reader as the selection moves. */
export function sayPart(landing: Landing, part: Part): string {
  const selected = selectedOf(part);
  if (selected.kind === "link") {
    const worker = landing.nodes.find(({ id }) => id === selected.worker);
    return `The link between ${landing.controller} and ${worker?.name ?? selected.worker}`;
  }
  const node = landing.nodes.find(({ id }) => id === selected.node);
  if (node === undefined) {
    return "";
  }
  if (selected.kind === "node") {
    return `${node.name}, ${node.plaque}`;
  }
  const chip = node.roles.find(({ role }) => role === selected.role);
  return chip === undefined ? "" : `${chip.words} on ${node.name}`;
}
