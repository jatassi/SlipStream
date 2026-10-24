import "../styles.css";

import { useState } from "react";
import { expect, onTestFinished, test, vi } from "vitest";
import { commands, page, userEvent } from "vitest/browser";

import { landingOf } from "../routes/_app/nodes/-landing/view";
import { renderKit } from "../test/render-kit";
import type { Drop } from "./drag";
import { STAGE_WIDTH } from "./drawing";
import { DiagramFigure } from "./figure";
import { fixtureClock, fixtureOf } from "./fixtures";
import type { Part } from "./parts-order";

const fixture = fixtureOf(3);
const landing = landingOf(fixture.topology, fixture.board, new Set(), fixtureClock);
const [home, left, right] = landing.nodes.map(({ id }) => id);

const tile = (node: string | undefined) => `[data-node="${String(node)}"] [data-item="tile.base"]`;
const chip = (node: string | undefined, role: string) =>
  `[data-node="${String(node)}"] [data-chip="${role}"]`;
const zoomButton = (name: string) => page.getByRole("button", { name, exact: true });

async function render(takesDrags = true) {
  const selected = vi.fn<(part: Part | undefined) => void>();
  const dropped = vi.fn<(drop: Drop) => void>();
  function Figure() {
    const [part, setPart] = useState<Part>();
    return (
      <DiagramFigure
        landing={landing}
        takesDrags={takesDrags}
        selected={part}
        onSelect={(next) => {
          selected(next);
          setPart(next);
        }}
        gestureOf={(role, from, to) => ({ role, from, to, verdict: "allow", word: "Moves" })}
        onDrop={dropped}
      />
    );
  }
  renderKit(<Figure />);
  await expect.element(page.getByRole("figure", { name: "Diagram" })).toBeVisible();
  return { selected, dropped };
}

function svg(): SVGSVGElement {
  const drawn = document.querySelector<SVGSVGElement>(".diagram svg");
  if (drawn === null) {
    throw new Error("the Diagram drew no svg");
  }
  return drawn;
}

const view = (): number[] => (svg().getAttribute("viewBox") ?? "").split(" ").map(Number);
const zoom = (): number => STAGE_WIDTH / (view()[2] ?? 0);

function boxOf(selector: string): DOMRect {
  const element = document.querySelector(selector);
  if (element === null) {
    throw new Error(`${selector} is not drawn`);
  }
  return element.getBoundingClientRect();
}

const centreOf = (selector: string): readonly [number, number] => {
  const { x, y, width, height } = boxOf(selector);
  return [x + width / 2, y + height / 2];
};

const nodeUnder = ([x, y]: readonly [number, number]) =>
  document.elementFromPoint(x, y)?.closest<SVGElement>("[data-node]")?.dataset["node"];

/** Turns the mouse wheel over a point on screen. */
function wheel([x, y]: readonly [number, number], deltaY: number): Promise<void> {
  const figure = boxOf(".diagram");
  return commands.wheel(
    { selector: ".diagram", at: [(x - figure.x) / figure.width, (y - figure.y) / figure.height] },
    deltaY,
  );
}

async function reducedMotion() {
  await commands.emulateReducedMotion(true);
  onTestFinished(() => commands.emulateReducedMotion(false));
}

test("the Diagram opens on the fitted view, and the wheel zooms about the pointer, keeping the tile under it there", async () => {
  await render();
  expect(view().slice(0, 3)).toEqual([0, 0, STAGE_WIDTH]);
  await expect.element(zoomButton("Zoom out")).toBeDisabled();
  await expect.element(zoomButton("Fit")).toBeDisabled();
  const at = centreOf(tile(right));
  const before = boxOf(tile(right));
  await wheel(at, -300);
  await expect.poll(zoom).toBeGreaterThan(1.5);
  expect(nodeUnder(at)).toBe(right);
  expect(boxOf(tile(right)).width).toBeGreaterThan(before.width * 1.5);
  await expect.element(zoomButton("Fit")).toBeEnabled();
});

function gesture(type: string, scale: number, [clientX, clientY]: readonly [number, number]) {
  const event = new UIEvent(type, { bubbles: true, cancelable: true });
  Object.defineProperties(event, {
    scale: { value: scale },
    clientX: { value: clientX },
    clientY: { value: clientY },
  });
  document.querySelector(".diagram")?.dispatchEvent(event);
  return event;
}

test("Safari's trackpad pinch zooms about the pointer and keeps the page from zooming", async () => {
  await render();
  await userEvent.hover(page.getByRole("figure", { name: "Diagram" }));
  const at = centreOf(tile(right));
  gesture("gesturestart", 1, at);
  gesture("gesturechange", 1.5, at);
  const last = gesture("gesturechange", 2, at);
  expect(last.defaultPrevented).toBe(true);
  await expect.poll(zoom).toBeCloseTo(2);
  expect(nodeUnder(at)).toBe(right);
});

test("Zoom in steps up to five times the fitted view, and Fit goes back to it", async () => {
  await reducedMotion();
  await render();
  await userEvent.click(zoomButton("Zoom in"));
  expect(zoom()).toBeCloseTo(1.5);
  await userEvent.click(zoomButton("Zoom out"));
  expect(zoom()).toBeCloseTo(1);
  for (const _ of [1, 2, 3, 4]) {
    // oxlint-disable-next-line no-await-in-loop -- each press steps from the last
    await userEvent.click(zoomButton("Zoom in"));
  }
  expect(zoom()).toBeCloseTo(5);
  await expect.element(zoomButton("Zoom in")).toBeDisabled();
  await userEvent.click(zoomButton("Fit"));
  expect(zoom()).toBeCloseTo(1);
  await expect.element(zoomButton("Fit")).toBeDisabled();
});

test("+, - and 0 zoom while the figure has focus, and the arrow keys still walk its parts", async () => {
  await reducedMotion();
  const { selected } = await render();
  document.querySelector<HTMLElement>(".diagram")?.focus();
  await userEvent.keyboard("+");
  expect(zoom()).toBeCloseTo(1.5);
  await userEvent.keyboard("{ArrowRight}");
  expect(selected).toHaveBeenLastCalledWith(`node:${String(home)}`);
  await userEvent.keyboard("+");
  await userEvent.keyboard("-");
  expect(zoom()).toBeCloseTo(1.5);
  await userEvent.keyboard("0");
  expect(zoom()).toBeCloseTo(1);
});

test("a press of Zoom in eases over several frames, and under reduced motion it jumps", async () => {
  await render();
  vi.useFakeTimers({ toFake: ["requestAnimationFrame", "cancelAnimationFrame"] });
  onTestFinished(() => {
    vi.useRealTimers();
  });
  await userEvent.click(zoomButton("Zoom in"));
  vi.advanceTimersToNextFrame();
  vi.advanceTimersToNextFrame();
  await expect.poll(zoom).toBeGreaterThan(1);
  expect(zoom()).toBeLessThan(1.5);
  vi.advanceTimersByTime(1000);
  await expect.poll(zoom).toBeCloseTo(1.5);
  await reducedMotion();
  await userEvent.click(zoomButton("Zoom in"));
  expect(zoom()).toBeCloseTo(2.25);
});

test("dragging the floor pans a zoomed view with the pointer, selects nothing, and does nothing at the fitted view", async () => {
  await reducedMotion();
  const { selected } = await render();
  const floor = { selector: ".diagram", at: [0.5, 0.97] } as const;
  const fit = svg().getAttribute("viewBox");
  await commands.swipe(floor, [-160, 0], false);
  expect(svg().getAttribute("viewBox")).toBe(fit);
  await userEvent.click(zoomButton("Zoom in"));
  await userEvent.click(zoomButton("Zoom in"));
  const before = boxOf(tile(home));
  await commands.swipe(floor, [-160, -60], false);
  const after = boxOf(tile(home));
  expect(after.x - before.x).toBeCloseTo(-160, 0);
  expect(after.y - before.y).toBeCloseTo(-60, 0);
  expect(selected).not.toHaveBeenCalled();
});

test("zoomed in, a Role chip still drags onto another tile and a click still selects the part it lands on", async () => {
  await reducedMotion();
  const { selected, dropped } = await render();
  const [hx, hy] = centreOf(tile(home));
  const [wx, wy] = centreOf(tile(right));
  await wheel([(hx + wx) / 2, (hy + wy) / 2], -150);
  await expect.poll(zoom).toBeGreaterThan(1.2);
  const zoomed = view();
  await commands.drag(chip(home, "downloading"), tile(right));
  expect(view()).toEqual(zoomed);
  expect(dropped).toHaveBeenCalledWith({
    role: "downloading",
    from: home,
    to: right,
    add: false,
  });
  await userEvent.click(page.elementLocator(document.querySelector(tile(right)) ?? document.body));
  expect(selected).toHaveBeenLastCalledWith(`node:${String(right)}`);
});

test("look-only, two fingers pinch the Diagram larger and one finger pans it without dragging a chip", async () => {
  const { dropped } = await render(false);
  await commands.pinch({ selector: tile(left), at: [0.5, 0.5] }, 80, 240);
  await expect.poll(zoom).toBeGreaterThan(2);
  const before = view();
  await commands.swipe({ selector: ".diagram", at: [0.5, 0.5] }, [-120, 0], true);
  await expect.poll(() => view()[0]).toBeGreaterThan(before[0] ?? 0);
  expect(dropped).not.toHaveBeenCalled();
});
