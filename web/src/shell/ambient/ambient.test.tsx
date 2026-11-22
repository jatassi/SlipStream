import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { expect, onTestFinished, test, vi } from "vitest";
import { cdp, commands, page } from "vitest/browser";

import { type Artwork, artworkHandlers, artworkUrl } from "../../test/ambient/fixtures";
import { worker } from "../../test/worker";
import { Ambient } from "./ambient";
import { type AmbientSource, useAmbient } from "./use-ambient";

function Page({ source }: { source: AmbientSource }) {
  useAmbient(source);
  return null;
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
}

const backdrop = async () =>
  page.elementLocator(
    await vi.waitUntil(() => document.querySelector('[data-ambient-backdrop="current"]')),
  );

async function prefer(name: string, value: string) {
  await cdp().send("Emulation.setEmulatedMedia", { features: [{ name, value }] });
  onTestFinished(async () => {
    await cdp().send("Emulation.setEmulatedMedia", { features: [] });
    // Clearing the features drops the suite's own no-preference too, and a
    // runner whose OS turns animations off would then report reduce.
    await commands.emulateReducedMotion(false);
  });
}

test("a Work's page draws its artwork behind the app", async () => {
  mount(
    <>
      <Ambient />
      <Page source={work("red")} />
    </>,
  );
  await expect.element(await backdrop()).toBeVisible();
});

test("reduced transparency hides the artwork", async () => {
  await prefer("prefers-reduced-transparency", "reduce");
  mount(
    <>
      <Ambient />
      <Page source={work("red")} />
    </>,
  );
  await expect.element(await backdrop()).not.toBeVisible();
});

test("reduced motion swaps the artwork in at once", async () => {
  await prefer("prefers-reduced-motion", "reduce");
  mount(
    <>
      <Ambient />
      <Page source={work("red")} />
    </>,
  );
  await backdrop();
  expect(document.getAnimations()).toEqual([]);
});
