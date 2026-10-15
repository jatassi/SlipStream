import type { ReactNode } from "react";

import { Wordmark } from "../ui/wordmark";

export function EntryPage({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-6 px-4 pt-[max(48px,env(safe-area-inset-top))] pb-[max(48px,env(safe-area-inset-bottom))]">
      <header className="w-full max-w-[400px] px-1 text-sentence">
        <Wordmark />
      </header>
      <main className="flex w-full max-w-[400px] flex-col gap-5 rounded-card bg-glass p-6 text-(length:--body-size) leading-(--body-line-height) text-ink-2 ring-1 ring-hairline ring-inset in-data-[density=touch]:p-5">
        <h1 className="text-title text-ink-1">{title}</h1>
        {children}
      </main>
    </div>
  );
}
