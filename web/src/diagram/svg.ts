const SVG = "http://www.w3.org/2000/svg";

/** Makes an SVG element with attributes and appends it to parent. */
export function make<K extends keyof SVGElementTagNameMap>(
  tag: K,
  attributes: Readonly<Record<string, string>>,
  parent: Element,
): SVGElementTagNameMap[K] {
  const element = document.createElementNS(SVG, tag);
  for (const [name, value] of Object.entries(attributes)) {
    element.setAttribute(name, value);
  }
  parent.append(element);
  return element;
}

export const fixed = (value: number): string => value.toFixed(2);

/** Hides a pooled element once, until it is shown again. */
export function hide(pooled: { shown: boolean }, element: Element) {
  if (pooled.shown) {
    element.setAttribute("opacity", "0");
    pooled.shown = false;
  }
}

/** Writes an attribute only when its value changes. */
export function write(
  written: WeakMap<Element, Map<string, string>>,
  element: Element | null,
  name: string,
  value: string,
) {
  if (element === null) {
    return;
  }
  const values = written.get(element) ?? new Map<string, string>();
  if (values.get(name) !== value) {
    values.set(name, value);
    written.set(element, values);
    element.setAttribute(name, value);
  }
}
