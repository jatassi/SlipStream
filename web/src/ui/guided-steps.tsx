import { LinkSquare02Icon, Tick02Icon } from "@hugeicons/core-free-icons";
import { type ReactNode, useState } from "react";

import { Icon } from "./icon";

export type GuidedStep = {
  readonly label: string;
  /** The real page the step happens on. Following it ticks the step. */
  readonly link?: { readonly label: string; readonly href: string };
  readonly done?: boolean;
  /** What the step asks for in place, such as the field a key is pasted into. */
  readonly children?: ReactNode;
};

const discLook = {
  done: "bg-selected text-ink-1",
  current: "bg-solid text-on-solid shadow-live-glow",
  pending: "bg-fill text-ink-2 ring-1 ring-hairline-strong ring-inset",
} as const;

const linkLook =
  "inline-flex h-(--control-height) items-center justify-center gap-2 self-start rounded-capsule bg-fill px-3.5 text-button font-semibold text-ink-1 ring-1 ring-hairline-strong ring-inset hover:bg-fill-hover in-data-[density=touch]:self-stretch in-data-[density=touch]:text-body-touch";

/**
 * A numbered list of steps on a connector. The first step not done is
 * current, and following a step's link ticks it.
 */
export function GuidedSteps({ label, steps }: { label: string; steps: readonly GuidedStep[] }) {
  const [followed, setFollowed] = useState<ReadonlySet<number>>(new Set());
  const done = steps.map((step, index) => step.done === true || followed.has(index));
  const current = done.indexOf(false);
  return (
    <ol aria-label={label} className="flex flex-col">
      {steps.map((step, index) => {
        const state = done[index] === true ? "done" : index === current ? "current" : "pending";
        return (
          <li
            key={step.label}
            aria-current={state === "current" ? "step" : undefined}
            className="relative flex gap-3 pb-5 last:pb-0 not-last:before:absolute not-last:before:top-8.5 not-last:before:bottom-1 not-last:before:left-[14.5px] not-last:before:w-px not-last:before:bg-hairline-strong"
          >
            <span
              aria-hidden="true"
              className={`flex size-7.5 shrink-0 items-center justify-center rounded-capsule text-meta font-semibold tabular-nums ${discLook[state]}`}
            >
              {state === "done" ? <Icon icon={Tick02Icon} className="size-4" /> : index + 1}
            </span>
            <div className="flex min-w-0 flex-1 flex-col gap-2 pt-1">
              <p className={state === "pending" ? "text-ink-2" : "text-ink-1"}>
                <span className="sr-only">
                  {`Step ${String(index + 1)}${state === "done" ? ", done" : ""}: `}
                </span>
                {step.label}
              </p>
              {step.link !== undefined && (
                <a
                  href={step.link.href}
                  target="_blank"
                  rel="noreferrer"
                  className={linkLook}
                  onClick={() => {
                    setFollowed((now) => new Set([...now, index]));
                  }}
                >
                  {step.link.label}
                  <Icon icon={LinkSquare02Icon} className="size-4" />
                </a>
              )}
              {step.children}
            </div>
          </li>
        );
      })}
    </ol>
  );
}
