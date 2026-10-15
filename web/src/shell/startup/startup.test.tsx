import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { commands, page, userEvent } from "vitest/browser";

import { renderKit } from "../../test/render-kit";
import { writePref } from "../prefs";
import { Startup, markTransitionName, useMarkTransitionName } from "./startup";

const sequence = page.getByRole("img", { name: "SlipStream" });

const nextFrame = () =>
  new Promise((resolve) => {
    requestAnimationFrame(resolve);
  });

const drawing = async () => {
  await expect.poll(() => document.fonts.check('600 16px "Geist Variable"')).toBe(true);
  await nextFrame();
  await nextFrame();
  return sequence.element().innerHTML;
};

const pause = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

beforeEach(() => {
  writePref("startupPlayed", false);
  writePref("startupOff", false);
});

afterEach(async () => {
  await commands.emulateReducedMotion(false);
});

test("a new tab plays the sequence and a key press skips it", async () => {
  renderKit(<Startup handOff="fade" />);

  await expect.element(sequence).toBeVisible();
  const first = await drawing();
  await pause(150);
  expect(await drawing()).not.toBe(first);

  await userEvent.keyboard("{Enter}");
  await expect.element(sequence).not.toBeInTheDocument();
});

test("the sequence leaves on its own after 4.4s", async () => {
  renderKit(<Startup handOff="fade" />);

  await expect.element(sequence).toBeVisible();
  await pause(3000);
  await expect.element(sequence).toBeVisible();
  await expect.element(sequence, { timeout: 3000 }).not.toBeInTheDocument();
});

test("a second render in the same tab doesn't replay it", async () => {
  renderKit(<Startup handOff="fade" />);
  await expect.element(sequence).toBeVisible();

  renderKit(
    <>
      <Startup handOff="fade" />
      <p>Second</p>
    </>,
  );
  await expect.element(page.getByText("Second")).toBeVisible();
  expect(page.getByRole("img", { name: "SlipStream" }).all()).toHaveLength(1);
});

test("the sequence doesn't play when it's off in this browser", async () => {
  writePref("startupOff", true);
  renderKit(
    <>
      <Startup handOff="fade" />
      <p>Beneath</p>
    </>,
  );

  await expect.element(page.getByText("Beneath")).toBeVisible();
  expect(sequence.query()).toBeNull();
});

test("under reduced motion the finished lockup holds still, then leaves", async () => {
  await commands.emulateReducedMotion(true);
  renderKit(<Startup handOff="view-transition" />);

  await expect.element(sequence).toBeVisible();
  const first = await drawing();
  await pause(150);
  expect(await drawing()).toBe(first);
  await expect.element(sequence, { timeout: 1500 }).not.toBeInTheDocument();
});

function SidebarMark() {
  const name = useMarkTransitionName();
  return (
    <div data-testid="sidebar-mark" style={name === undefined ? {} : { viewTransitionName: name }}>
      Mark
    </div>
  );
}

test("in the desktop shell the mark settles into the element named for it", async () => {
  const transitions: ViewTransition[] = [];
  const start = document.startViewTransition.bind(document);
  vi.spyOn(document, "startViewTransition").mockImplementation((options) => {
    const transition = start(options);
    transitions.push(transition);
    return transition;
  });
  renderKit(
    <>
      <Startup handOff="view-transition" />
      <SidebarMark />
    </>,
  );
  await expect.element(sequence).toBeVisible();
  await drawing();

  await userEvent.keyboard("{Enter}");
  await expect.poll(() => transitions.length).toBe(1);
  const [transition] = transitions;
  await transition?.ready;
  const pseudos = document
    .getAnimations()
    .map(({ effect }) => (effect instanceof KeyframeEffect ? effect.pseudoElement : null));
  expect(pseudos).toEqual(
    expect.arrayContaining([
      `::view-transition-group(${markTransitionName})`,
      `::view-transition-old(${markTransitionName})`,
      `::view-transition-new(${markTransitionName})`,
    ]),
  );
  await transition?.finished;
  expect(sequence.query()).toBeNull();
  await expect
    .poll(() => getComputedStyle(page.getByTestId("sidebar-mark").element()).viewTransitionName)
    .toBe("none");
});
