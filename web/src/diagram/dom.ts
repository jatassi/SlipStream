import type { RoleName } from "../nodes/roles";
import { type Part, isPart } from "./parts-order";

// Reading and marking the Diagram's drawn parts by their data attributes.

const svgOf = (element: Element): SVGElement | undefined =>
  element instanceof SVGElement ? element : undefined;

/** The part under an event's target, when it is one. */
export function partAt(target: EventTarget | null): Part | undefined {
  const part =
    target instanceof Element
      ? svgOf(target)?.closest<SVGElement>("[data-part]")?.dataset["part"]
      : undefined;
  return part !== undefined && isPart(part) ? part : undefined;
}

/** The Node whose tile is at a point on the screen. */
export const nodeAt = (x: number, y: number): string | undefined =>
  document.elementFromPoint(x, y)?.closest<SVGElement>("[data-node]")?.dataset["node"];

/** The Role chip a pointer went down on, and the Node it is on. */
export function chipAt(
  target: EventTarget | null,
  roleOf: (name: string) => RoleName | undefined,
): { readonly role: RoleName; readonly from: string } | undefined {
  const chip = target instanceof Element ? target.closest<SVGElement>("[data-chip]") : null;
  const from = chip?.closest<SVGElement>("[data-node]")?.dataset["node"];
  const role = roleOf(chip?.dataset["chip"] ?? "");
  return from === undefined || role === undefined ? undefined : { role, from };
}

/** Lights one part, dimming the rest; nothing lit when part is undefined. */
export function lightPart(svg: SVGSVGElement, figure: HTMLElement, part: Part | undefined) {
  for (const on of svg.querySelectorAll<SVGElement>("[data-on]")) {
    delete on.dataset["on"];
  }
  if (part === undefined) {
    delete figure.dataset["focus"];
    return;
  }
  figure.dataset["focus"] = part;
  for (const element of svg.querySelectorAll<SVGElement>(`[data-part="${CSS.escape(part)}"]`)) {
    element.dataset["on"] = "";
  }
}

/** Tints the tile a chip is dragged over with the plan's verdict on that drop. */
export function markVerdict(
  svg: SVGSVGElement,
  node: string | undefined,
  verdict: string | undefined,
) {
  for (const tile of svg.querySelectorAll<SVGElement>("[data-verdict]")) {
    delete tile.dataset["verdict"];
  }
  const tile =
    node === undefined ? null : svg.querySelector<SVGElement>(`[data-node="${CSS.escape(node)}"]`);
  if (tile !== null && verdict !== undefined) {
    tile.dataset["verdict"] = verdict;
  }
}
