import { Mark } from "./mark";

export function Wordmark({
  viewTransitionName,
  className = "",
}: {
  viewTransitionName?: string;
  className?: string;
}) {
  return (
    <span
      className={`inline-flex items-center gap-[calc(1.4em/3)] tracking-[-0.03em] text-ink-1 ${className}`}
      style={viewTransitionName === undefined ? undefined : { viewTransitionName }}
    >
      <Mark decorative className="size-[1.4em]" />
      <span>
        <span className="font-semibold">Slip</span>
        <span className="font-normal">Stream</span>
      </span>
    </span>
  );
}
