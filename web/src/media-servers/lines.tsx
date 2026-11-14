import type { MediaserversStateLevel as Level } from "../api/gen/model";
import { Icon } from "../ui/icon";
import { signals } from "../ui/signal-chip";

const inks: { readonly [L in Level]: string } = {
  ok: "text-ink-1",
  warn: "text-warn",
  fail: "text-fail-ink",
};

/** One sentence led by its signal's icon. */
export function SignalLine({ level, text }: { level: Level; text: string }) {
  return (
    <p className={`flex items-start gap-1.5 text-pretty ${inks[level]}`}>
      <Icon icon={signals[level].icon} className="mt-[0.2em] size-4 shrink-0" />
      {text}
    </p>
  );
}

export const WarnLine = ({ text }: { text: string }) => <SignalLine level="warn" text={text} />;
