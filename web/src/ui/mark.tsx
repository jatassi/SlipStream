export function Mark({
  size = 24,
  decorative = false,
  viewTransitionName,
  className = "",
}: {
  size?: number;
  decorative?: boolean;
  viewTransitionName?: string;
  className?: string;
}) {
  return (
    <svg
      viewBox="0 0 64 64"
      width={size}
      height={size}
      fill="none"
      className={`shrink-0 ${className}`}
      style={viewTransitionName === undefined ? undefined : { viewTransitionName }}
      {...(decorative ? { "aria-hidden": true } : { role: "img", "aria-label": "SlipStream" })}
    >
      <path
        d="M40 14H25a9 9 0 0 0 0 18h14a9 9 0 0 1 0 18H24"
        strokeWidth="8"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="stroke-ink-1"
      />
      <circle cx="52" cy="14" r="5.5" className="fill-glow" />
      <circle cx="12" cy="50" r="5.5" className="fill-glow" />
    </svg>
  );
}
