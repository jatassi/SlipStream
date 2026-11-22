import { Fragment, type ReactNode } from "react";

import { Icon } from "./icon";
import { LiveDot } from "./live-chip";
import { useShell } from "./shell-context";
import { signals } from "./signal-chip";

export type StatusPart =
  | { kind: "live"; text: string }
  /** `words` draws the text with a part inside it, such as a defined term. */
  | { kind: "plain"; text: string; words?: ReactNode }
  | { kind: "failure"; text: string; href: string }
  | { kind: "warning"; text: string; href: string };

const signalLink = {
  failure: { icon: signals.fail.icon, ink: "text-fail-ink" },
  warning: { icon: signals.warn.icon, ink: "text-warn" },
} as const;

function Part({ part }: { part: StatusPart }) {
  if (part.kind === "live") {
    return (
      <span>
        <span className="mr-2.5 inline-flex align-[0.2em]">
          <LiveDot />
        </span>
        {part.text}
      </span>
    );
  }
  if (part.kind === "plain") {
    return <span>{part.words ?? part.text}</span>;
  }
  const { icon, ink } = signalLink[part.kind];
  return (
    <a href={part.href} className={`rounded-item underline-offset-4 hover:underline ${ink}`}>
      <Icon icon={icon} className="mr-1.5 inline-block size-[0.8em] align-[-0.05em]" />
      {part.text}
    </a>
  );
}

/** `section` sets the sentence at a card title's size, for a page that leads with another. */
export function StatusSentence({
  parts,
  scale = "page",
}: {
  parts: readonly StatusPart[];
  scale?: "page" | "section";
}) {
  const phone = useShell() === "phone";
  const type =
    scale === "section"
      ? "text-card-title font-semibold"
      : phone
        ? "text-sentence-touch"
        : "text-sentence";
  return (
    <p className={`${type} text-ink-1 tabular-nums text-shadow-[0_1px_16px_rgb(0_0_0/.45)]`}>
      {parts.map((part, index) => (
        <Fragment key={`${part.kind}:${part.text}`}>
          {index > 0 && (
            <>
              {" "}
              <span aria-hidden="true" className="px-1 text-ink-4">
                ·
              </span>{" "}
            </>
          )}
          <Part part={part} />
        </Fragment>
      ))}
    </p>
  );
}
