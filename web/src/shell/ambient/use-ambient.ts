import { useEffect, useState } from "react";

import { neutralTint } from "../../theme.gen";
import { sampleImage, type Tint } from "./sampler";

export type AmbientSource =
  | { readonly kind: "neutral" }
  | { readonly kind: "artwork"; readonly url: string; readonly workId: string };

export type AmbientOptions = {
  /** How long the ambient holds this source after its caller unmounts. */
  readonly returnDelay?: number;
};

export type Backdrop = {
  readonly id: number;
  readonly url: string;
  readonly brightness: number;
  readonly leaving: boolean;
};

/** What the ambient shows: the tint, and the backdrop arriving over the one leaving. */
export type Scene = { readonly tint: Tint; readonly backdrops: readonly Backdrop[] };

const neutral: AmbientSource = { kind: "neutral" };

// Callers keyed by render order: a parent renders before its children, so
// the deepest caller holds the highest order.
const callers = new Map<number, AmbientSource>();
const releases = new Map<number, ReturnType<typeof setTimeout>>();
const listeners = new Set<() => void>();
let lastOrder = 0;
let lastBackdrop = 0;
let active: AmbientSource = neutral;
let scene: Scene = { tint: neutralTint, backdrops: [] };

export function subscribeScene(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function ambientScene() {
  return scene;
}

/**
 * Sets the ambient for as long as the calling component is mounted. The
 * deepest caller wins, and with none the ambient is neutral.
 */
export function useAmbient(source: AmbientSource, { returnDelay = 0 }: AmbientOptions = {}) {
  const [order] = useState(nextOrder);
  useEffect(() => {
    callers.set(order, source);
    settle();
  }, [order, source]);
  // The release waits on a timer even with no delay, so a remount or a new
  // delay cancels it instead of dropping a caller that is still mounted.
  useEffect(() => {
    clearTimeout(releases.get(order));
    return () => {
      releases.set(
        order,
        setTimeout(() => {
          releases.delete(order);
          callers.delete(order);
          settle();
        }, returnDelay),
      );
    };
  }, [order, returnDelay]);
}

function nextOrder() {
  lastOrder += 1;
  return lastOrder;
}

function settle() {
  const next = callers.get(Math.max(0, ...callers.keys())) ?? neutral;
  if (same(next, active)) {
    return;
  }
  active = next;
  void show(next);
}

async function show(shown: AmbientSource) {
  if (shown.kind === "neutral") {
    publish({ tint: neutralTint, backdrops: leave(scene.backdrops) });
    return;
  }
  try {
    const sample = await sampleImage(shown.url);
    if (shown !== active) {
      return;
    }
    lastBackdrop += 1;
    const arriving = {
      id: lastBackdrop,
      url: shown.url,
      brightness: sample.brightness,
      leaving: false,
    };
    publish({ tint: sample, backdrops: [...leave(scene.backdrops), arriving] });
  } catch {
    if (shown === active) {
      publish({ tint: neutralTint, backdrops: leave(scene.backdrops) });
    }
  }
}

function leave(backdrops: readonly Backdrop[]): Backdrop[] {
  return backdrops
    .slice(-1)
    .map(({ id, url, brightness }) => ({ id, url, brightness, leaving: true }));
}

function publish(next: Scene) {
  scene = next;
  for (const listener of listeners) {
    listener();
  }
}

function same(a: AmbientSource, b: AmbientSource) {
  if (a.kind === "neutral" || b.kind === "neutral") {
    return a.kind === b.kind;
  }
  return a.url === b.url && a.workId === b.workId;
}
