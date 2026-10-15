import { Cancel01Icon } from "@hugeicons/core-free-icons";
import { type ReactNode, useId } from "react";

import { Icon } from "./icon";
import "./rows.css";

export function InspectorSection({ title, children }: { title?: string; children: ReactNode }) {
  return (
    <section className="rounded-card bg-(--inspector-well) p-4">
      {title !== undefined && <h3 className="mb-3 text-heading text-ink-1">{title}</h3>}
      {children}
    </section>
  );
}

export function InspectorBody({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-3">{children}</div>;
}

export function Inspector({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
}) {
  const titleId = useId();
  return (
    <aside
      aria-labelledby={titleId}
      className="flex h-full w-[min(480px,40vw)] flex-col rounded-dialog bg-popover text-(length:--body-size) leading-(--body-line-height) text-ink-1 shadow-float-large backdrop-blur-[30px] backdrop-saturate-[1.7] transition-[translate,opacity] duration-(--duration-inspector) ease-standard starting:translate-x-4 starting:opacity-0 motion-reduce:transition-opacity motion-reduce:starting:translate-x-0"
    >
      <header className="flex items-center gap-3 px-5 pt-4 pb-3">
        <h2 id={titleId} className="min-w-0 flex-1 truncate text-title">
          {title}
        </h2>
        <button
          type="button"
          aria-label="Close"
          onClick={onClose}
          className="inline-flex size-(--icon-button-size) items-center justify-center rounded-capsule text-ink-2 transition-colors duration-(--duration-control) ease-standard hover:bg-fill hover:text-ink-1"
        >
          <Icon icon={Cancel01Icon} className="size-4" />
        </button>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-5">
        <InspectorBody>{children}</InspectorBody>
      </div>
    </aside>
  );
}
