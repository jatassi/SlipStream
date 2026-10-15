export function Spinner({ label, className = "" }: { label?: string; className?: string }) {
  return (
    <svg
      viewBox="0 0 14 14"
      fill="none"
      strokeWidth="2"
      className={`size-3.5 shrink-0 animate-spinner stroke-current motion-reduce:animate-none ${className}`}
      {...(label === undefined ? { "aria-hidden": true } : { role: "status", "aria-label": label })}
    >
      <circle cx="7" cy="7" r="6" className="opacity-20" />
      <path d="M7 1a6 6 0 0 1 6 6" strokeLinecap="round" />
    </svg>
  );
}
