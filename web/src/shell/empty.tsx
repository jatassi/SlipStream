import type { ReactNode } from "react";

export function Empty({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex max-w-[560px] flex-col items-start gap-4 rounded-card bg-glass p-5 ring-1 ring-hairline ring-inset">
      <p className="text-(length:--body-size) leading-(--body-line-height) text-ink-1">
        {children}
      </p>
      {action}
    </div>
  );
}
