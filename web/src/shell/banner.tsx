import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import { ContinueWithout } from "../reconciling/continue";
import { useBanner } from "../reconciling/use-banner";
import { type Next, bannerLead } from "../reconciling/words";
import { Button } from "../ui/button";
import { Icon } from "../ui/icon";
import { useShell } from "../ui/shell-context";
import { Spinner } from "../ui/spinner";

function DesktopBanner({ next }: { next: Next | "loading" }) {
  const [only] = next !== "loading" && next.kind === "offline" ? next.nodes : [];
  return (
    <aside
      aria-label="Reconciling"
      className="flex flex-wrap items-center gap-x-6 gap-y-3 bg-glass px-7 py-3 text-body shadow-[inset_0_-1px_0_var(--color-hairline)]"
    >
      <Spinner />
      <div className="flex min-w-0 flex-1 basis-[420px] flex-col gap-0.5">
        <p className="text-ink-1">{bannerLead}</p>
        {next !== "loading" && <p className="text-ink-2">{next.sentence}</p>}
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {only !== undefined && <ContinueWithout node={only} look="primary" />}
        <Button size="small" nativeButton={false} render={<Link to="/reconciling" />}>
          Open Reconciling
        </Button>
      </div>
    </aside>
  );
}

// On phone the line sticks above the tab bar at the end of the content, so
// it never covers the page's last rows.
function PhoneBanner({ next }: { next: Next | "loading" }) {
  return (
    <aside
      aria-label="Reconciling"
      className="sticky bottom-[calc(env(safe-area-inset-bottom)+57px)] z-10 mt-6"
    >
      <Link
        to="/reconciling"
        className="flex items-center gap-3 rounded-card bg-chrome px-4 py-3 shadow-float ring-1 ring-hairline ring-inset backdrop-blur-[24px] backdrop-saturate-[1.6]"
      >
        <Spinner />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5 text-meta">
          <span className="text-ink-1">{bannerLead}</span>
          {next !== "loading" && <span className="text-ink-2">{next.sentence}</span>}
        </span>
        <Icon icon={ArrowRight01Icon} className="size-5 shrink-0 text-ink-3" />
      </Link>
    </aside>
  );
}

const banners = { desktop: DesktopBanner, phone: PhoneBanner } as const;

/** The shell's line while a restore reconciles: what SlipStream is doing, then the next step. */
export function ReconcilingBanner() {
  const shown = useBanner();
  const Banner = banners[useShell()];
  return shown === undefined ? null : <Banner next={shown} />;
}
