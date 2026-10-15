import type { ReactNode } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";
import { expect, onTestFinished, test, vi } from "vitest";
import { cdp } from "vitest/browser";

import { type Artwork, artworkHandlers, artworkUrl } from "../../test/ambient/fixtures";
import { worker } from "../../test/worker";
import { Ambient } from "./ambient";
import { type AmbientOptions, type AmbientSource, useAmbient } from "./use-ambient";

function Page({
  source,
  options,
  children,
}: {
  source: AmbientSource;
  options?: AmbientOptions;
  children?: ReactNode;
}) {
  useAmbient(source, options);
  return children;
}

const work = (name: Artwork): AmbientSource => ({
  kind: "artwork",
  url: artworkUrl(name),
  workId: name,
});

function mount(ui: ReactNode) {
  worker.use(...artworkHandlers);
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  root.render(ui);
  onTestFinished(async () => {
    root.unmount();
    container.remove();
    await new Promise((resolve) => {
      setTimeout(resolve, 0);
    });
    for (const animation of document.documentElement.getAnimations()) {
      animation.finish();
    }
  });
  return root;
}

const transitions = () =>
  document.documentElement
    .getAnimations()
    .filter((animation) => animation instanceof CSSTransition)
    .toSorted((a, b) => a.transitionProperty.localeCompare(b.transitionProperty));

const tint = () => {
  const style = getComputedStyle(document.documentElement);
  return { h: style.getPropertyValue("--h"), s: style.getPropertyValue("--s") };
};

const backdrop = () => {
  const element = document.querySelector('[data-ambient-backdrop="current"]');
  return element === null ? null : getComputedStyle(element);
};

const backdropFilter = () => backdrop()?.filter ?? null;

async function prefer(name: string, value: string) {
  await cdp().send("Emulation.setEmulatedMedia", { features: [{ name, value }] });
  onTestFinished(async () => {
    await cdp().send("Emulation.setEmulatedMedia", { features: [] });
  });
}

test("a page without a Work sits on the neutral", async () => {
  mount(
    <>
      <Ambient />
      <Page source={{ kind: "neutral" }} />
    </>,
  );
  await expect.poll(tint).toEqual({ h: "214", s: "14%" });
  expect(backdropFilter()).toBeNull();
});

test.each([
  { name: "white", h: "220", s: "12%", filter: "blur(64px) saturate(1.8) brightness(0.255)" },
  { name: "black", h: "220", s: "12%", filter: "blur(64px) saturate(1.8) brightness(1)" },
  // Red is hue 0; from the neutral's 214 the short way round reaches it at 360.
  { name: "red", h: "360", s: "70%", filter: "blur(64px) saturate(1.8) brightness(0.536)" },
  { name: "two-tone", h: "221", s: "53%", filter: "blur(64px) saturate(1.8) brightness(0.255)" },
] as const)(
  "a Work with the $name artwork tints the page $h / $s",
  async ({ name, h, s, filter }) => {
    mount(
      <>
        <Ambient />
        <Page source={work(name)} />
      </>,
    );
    await expect.poll(backdropFilter).toBe(filter);
    await expect.poll(tint).toEqual({ h, s });
  },
);

test("the deepest caller wins", async () => {
  mount(
    <>
      <Ambient />
      <Page source={work("white")}>
        <Page source={work("red")} />
      </Page>
    </>,
  );
  await expect.poll(backdropFilter).toBe("blur(64px) saturate(1.8) brightness(0.536)");
  await expect.poll(tint).toEqual({ h: "360", s: "70%" });
});

test("leaving the Work's page returns to the neutral", async () => {
  const root = mount(
    <>
      <Ambient />
      <Page source={work("red")} />
    </>,
  );
  await expect.poll(tint).toEqual({ h: "360", s: "70%" });
  root.render(<Ambient />);
  await expect.poll(tint).toEqual({ h: "214", s: "14%" });
  expect(backdropFilter()).toBeNull();
});

test("a return delay holds the Work's tint after its page leaves", async () => {
  const root = mount(
    <>
      <Ambient />
      <Page source={work("red")} options={{ returnDelay: 450 }} />
    </>,
  );
  await expect.poll(tint).toEqual({ h: "360", s: "70%" });
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  onTestFinished(() => {
    vi.useRealTimers();
  });
  flushSync(() => {
    root.render(<Ambient />);
  });
  await vi.advanceTimersByTimeAsync(449);
  expect(tint()).toEqual({ h: "360", s: "70%" });
  await vi.advanceTimersByTimeAsync(1);
  vi.useRealTimers();
  await expect.poll(tint).toEqual({ h: "214", s: "14%" });
});

test("the registered --h and --s animate between tints", async () => {
  const root = mount(
    <>
      <Ambient />
      <Page source={work("red")} />
    </>,
  );
  await expect.poll(tint).toEqual({ h: "360", s: "70%" });
  root.render(<Ambient />);
  await expect.poll(() => transitions().map((t) => t.transitionProperty)).toEqual(["--h", "--s"]);
  for (const transition of transitions()) {
    transition.pause();
    transition.currentTime = 550;
  }
  expect(tint()).toEqual({ h: "218.12", s: "15.5804%" });
});

test("reduced transparency hides the backdrop and keeps the tint", async () => {
  await prefer("prefers-reduced-transparency", "reduce");
  mount(
    <>
      <Ambient />
      <Page source={work("red")} />
    </>,
  );
  await expect.poll(() => backdrop()?.display).toBe("none");
  await expect.poll(tint).toEqual({ h: "360", s: "70%" });
});

test("reduced motion swaps the tint and backdrop at once", async () => {
  await prefer("prefers-reduced-motion", "reduce");
  mount(
    <>
      <Ambient />
      <Page source={work("red")} />
    </>,
  );
  await expect.poll(tint).not.toEqual({ h: "214", s: "14%" });
  expect(tint()).toEqual({ h: "360", s: "70%" });
  expect(document.getAnimations()).toEqual([]);
});
