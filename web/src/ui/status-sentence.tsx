import { Alert02Icon, AlertCircleIcon } from "@hugeicons/core-free-icons";
import { Fragment } from "react";

import { Icon } from "./icon";
import "./rows.css";
import { useShell } from "./shell-context";

export type StatusPart =
  | { kind: "live"; text: string }
  | { kind: "plain"; text: string }
  | { kind: "failure"; text: string; href: string }
  | { kind: "warning"; text: string; href: string };

const signalLink = {
  failure: { icon: AlertCircleIcon, ink: "text-fail-ink" },
  warning: { icon: Alert02Icon, ink: "text-warn" },
} as const;

function Part({ part }: { part: StatusPart }) {
  if (part.kind === "live") {
    return (
      <span>
        <span
          aria-hidden="true"
          className="mr-2.5 inline-block size-2 rounded-full align-[0.15em] bg-solid shadow-[var(--shadow-live-glow),0_0_0_3px_var(--live-ring)]"
        />
        {part.text}
      </span>
    );
  }
  if (part.kind === "plain") {
    return <span>{part.text}</span>;
  }
  const { icon, ink } = signalLink[part.kind];
  return (
    <a href={part.href} className={`rounded-item underline-offset-4 hover:underline ${ink}`}>
      <Icon icon={icon} className="mr-1.5 inline-block size-[0.8em] align-[-0.05em]" />
      {part.text}
    </a>
  );
}

export function StatusSentence({ parts }: { parts: readonly StatusPart[] }) {
  const type = useShell() === "phone" ? "text-sentence-touch" : "text-sentence";
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
