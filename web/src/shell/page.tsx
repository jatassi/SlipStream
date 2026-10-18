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

/**
 * `beside` sits on the title's line in the desktop shell, such as the tabs
 * that switch the page's view. On phone it goes under the large title, or
 * heads the content of a pushed page.
 */
export function Page({
  title,
  beside,
  action,
  children,
}: {
  title?: string;
  beside?: ReactNode;
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
        beside={beside}
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
  beside,
  action,
  actionSlot,
}: {
  heading: Heading;
  title: string | undefined;
  beside: ReactNode;
  action: ReactNode;
  actionSlot: HTMLElement | null;
}) {
  if (heading.kind === "content") {
    return (
      <>
        {action !== undefined && actionSlot !== null && createPortal(action, actionSlot)}
        <div className="mb-6 flex flex-wrap items-center gap-x-6 gap-y-3">
          <h1 className="text-sentence text-ink-1">{title}</h1>
          {beside}
        </div>
      </>
    );
  }
  if (heading.kind === "large") {
    return (
      <>
        <div className="mt-1 mb-4 flex items-center justify-between gap-3">
          <LargeTitle title={title} onScrolledOut={heading.onScrolledOut} />
          {action}
        </div>
        {beside !== undefined && <div className="mb-4">{beside}</div>}
      </>
    );
  }
  if (beside === undefined && action === undefined) {
    return null;
  }
  return (
    <div className="mb-4 flex flex-wrap items-center gap-3">
      {beside}
      {action !== undefined && <div className="ml-auto">{action}</div>}
    </div>
  );
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
      (entries) => {
        onScrolledOut(entries.at(-1)?.isIntersecting === false);
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
