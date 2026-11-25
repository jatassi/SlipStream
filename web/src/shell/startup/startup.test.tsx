import { afterEach, beforeEach, expect, onTestFinished, test, vi } from "vitest";
import { commands, page, userEvent } from "vitest/browser";

import { renderKit } from "../../test/render-kit";
import { writePref } from "../prefs";
import { DURATION } from "../../../../brand/startup/sequence.js";
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

beforeEach(() => {
  writePref("startupPlayed", false);
  writePref("startupOff", false);
});

afterEach(async () => {
  await commands.emulateReducedMotion(false);
});

// The sequence's clock starts on the first frame after its fonts load, so
// the test runs the frames itself from there.
test("the sequence plays to its end, then leaves on its own", async () => {
  onTestFinished(() => {
    vi.useRealTimers();
  });
  vi.useFakeTimers({ toFake: ["requestAnimationFrame", "cancelAnimationFrame"] });
  renderKit(<Startup handOff="fade" />);

  await expect.poll(() => vi.getTimerCount()).toBe(1);
  await vi.advanceTimersByTimeAsync(DURATION * 1000 - 100);
  await expect.element(sequence).toBeVisible();
  await vi.advanceTimersByTimeAsync(200);
  await expect.element(sequence).not.toBeInTheDocument();
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

function SidebarMark() {
  const name = useMarkTransitionName();
  return <div style={name === undefined ? {} : { viewTransitionName: name }}>Mark</div>;
}

test("in the desktop shell the mark settles into the element named for it", async () => {
  const transitions: ViewTransition[] = [];
  const animated: Promise<(string | null)[]>[] = [];
  const start = document.startViewTransition.bind(document);
  vi.spyOn(document, "startViewTransition").mockImplementation((options) => {
    const transition = start(options);
    transitions.push(transition);
    // The pseudo-elements animate for 360ms, so they are read the moment the
    // transition starts rather than whenever the test next runs.
    animated.push(
      transition.ready.then(() =>
        document
          .getAnimations()
          .map(({ effect }) => (effect instanceof KeyframeEffect ? effect.pseudoElement : null)),
      ),
    );
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
  expect(await animated[0]).toEqual(
    expect.arrayContaining([
      `::view-transition-group(${markTransitionName})`,
      `::view-transition-old(${markTransitionName})`,
      `::view-transition-new(${markTransitionName})`,
    ]),
  );
  await transition?.finished;
  expect(sequence.query()).toBeNull();
});
