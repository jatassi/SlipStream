import type { NodesProbe } from "../../../api/gen/model";
import { Icon } from "../../../ui/icon";
import { signals } from "../../../ui/signal-chip";

/** What a probe that doesn't fit found. */
export type Finding = { readonly fit: "weak" | "fails"; readonly words: string };

export const findingsOf = (probes: readonly NodesProbe[]): Finding[] =>
  probes.flatMap(({ fit, words }) =>
    fit === "fits" || words === undefined ? [] : [{ fit, words }],
  );

const looks = {
  weak: { icon: signals.warn.icon, ink: "text-warn", word: "Weak" },
  fails: { icon: signals.fail.icon, ink: "text-fail-ink", word: "Fails" },
} as const;

/** The probes a Node runs on itself that found it weak or failing, each in words. */
export function Findings({ findings, label }: { findings: readonly Finding[]; label: string }) {
  if (findings.length === 0) {
    return null;
  }
  return (
    <ul aria-label={label} className="flex flex-col gap-1">
      {findings.map(({ fit, words }) => (
        <li key={words} className="flex items-start gap-1.5 text-meta text-ink-2">
          <Icon icon={looks[fit].icon} className={`mt-0.5 size-3.5 shrink-0 ${looks[fit].ink}`} />
          <span>
            <span className="sr-only">{looks[fit].word}: </span>
            {words}
          </span>
        </li>
      ))}
    </ul>
  );
}
