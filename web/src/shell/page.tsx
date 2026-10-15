import { type ReactNode, createContext, use, useEffect, useRef } from "react";
import { createPortal } from "react-dom";

import { useRouteTitle } from "./titles";

/**
 * Where a page's heading and main action go. In the desktop shell the heading
 * opens the content and the action sits in the top bar. On a phone's top-level
 * page they share the large-title row, and a pushed page's title is in the
 * nav bar with its action at the head of the content.
 */
export type Heading =
  | { readonly kind: "content" }
  | { readonly kind: "large"; readonly onScrolledOut: (out: boolean) => void }
  | { readonly kind: "bar" };

/**
 * The shell's places a page fills. The Inspector's column exists only in the
 * desktop shell; on phone an Inspector's content is a pushed page.
 */
export type ShellSlots = {
  readonly action: HTMLElement | null;
  readonly inspector: HTMLElement | null;
  readonly heading: Heading;
};

export const ShellSlotsContext = createContext<ShellSlots>({
  action: null,
  inspector: null,
  heading: { kind: "content" },
});

export const useShellSlots = (): ShellSlots => use(ShellSlotsContext);

export function Page({
  title,
  action,
  children,
}: {
  title?: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  const routeTitle = useRouteTitle();
  const slots = useShellSlots();
  return (
    <>
      <PageHead
        heading={slots.heading}
        title={title ?? routeTitle}
        action={action}
        actionSlot={slots.action}
      />
      {children}
    </>
  );
}

function PageHead({
  heading,
  title,
  action,
  actionSlot,
}: {
  heading: Heading;
  title: string | undefined;
  action: ReactNode;
  actionSlot: HTMLElement | null;
}) {
  if (heading.kind === "content") {
    return (
      <>
        {action !== undefined && actionSlot !== null && createPortal(action, actionSlot)}
        <h1 className="mb-6 text-sentence text-ink-1">{title}</h1>
      </>
    );
  }
  if (heading.kind === "large") {
    return (
      <div className="mt-1 mb-4 flex items-center justify-between gap-3">
        <LargeTitle title={title} onScrolledOut={heading.onScrolledOut} />
        {action}
      </div>
    );
  }
  return action === undefined ? null : <div className="mb-4 flex justify-end">{action}</div>;
}

// The large title hands its words to the nav bar once it scrolls under the
// bar's 44px.
function LargeTitle({
  title,
  onScrolledOut,
}: {
  title: string | undefined;
  onScrolledOut: (out: boolean) => void;
}) {
  const ref = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    const element = ref.current;
    if (element === null) {
      throw new Error("the large title mounted without its heading");
    }
    const observer = new IntersectionObserver(
      ([entry]) => {
        onScrolledOut(entry?.isIntersecting === false);
      },
      { rootMargin: "-44px 0px 0px 0px" },
    );
    observer.observe(element);
    return () => {
      observer.disconnect();
      onScrolledOut(false);
    };
  }, [onScrolledOut]);
  return (
    <h1 ref={ref} className="min-w-0 text-title-large text-ink-1">
      {title}
    </h1>
  );
}
