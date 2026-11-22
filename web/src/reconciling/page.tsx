import { useGetReconcilingSuspense } from "../api/gen/query/reconciling/reconciling";
import { Page } from "../shell/page";
import { useClock } from "../ui/clock";
import { FinishNow } from "./finish-now";
import { Lead } from "./lead";
import { NodesSection } from "./page-nodes";
import { ReviewSection, RootsSection } from "./page-sections";
import { UnknownSection } from "./page-unknown";

/**
 * The Reconciling page: what SlipStream waits on after a restore, each
 * Node and library root with the step that ends its wait, and the review
 * list. Outside a restore it lists only the Nodes that keep their torrents.
 */
export function ReconcilingPage() {
  const page = useGetReconcilingSuspense().data;
  const clock = useClock();
  return (
    <Page action={page.restore === null ? undefined : <FinishNow page={page} />}>
      <div className="mb-6">
        <Lead page={page} clock={clock} />
      </div>
      <NodesSection page={page} clock={clock} />
      <UnknownSection unknown={page.unknown} clock={clock} />
      <RootsSection page={page} />
      <ReviewSection page={page} />
    </Page>
  );
}
