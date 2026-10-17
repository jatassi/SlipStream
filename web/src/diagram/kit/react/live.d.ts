import type { RefObject } from "react";

export function asksForStillness(): boolean;
export function useStillness(): boolean;
export function useInView(ref: RefObject<Element | null>): boolean;
export function useReadout<T extends Element>(): {
  readonly anchor: RefObject<T | null>;
  readonly say: (text: string) => void;
};
export function useLoop(tick: (dt: number, calm: boolean) => boolean, running: boolean): () => void;
export function follow(
  current: number,
  target: number,
  dt: number,
  seconds: number,
  calm?: boolean,
): number;
