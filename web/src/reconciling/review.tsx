import { useGetReconcilingSuspense } from "../api/gen/query/reconciling/reconciling";
import { Page } from "../shell/page";
import { Empty } from "../ui/empty";
import { GroupSection } from "./review-group";

/**
 * The review list: recorded torrents no Node reported after the restore,
 * and torrents two Nodes hold, grouped by what their Nodes found of their
 * files.
 */
export function ReviewList() {
  const { review } = useGetReconcilingSuspense().data;
  const groups = review.filter(({ count }) => count > 0);
  return (
    <Page>
      {groups.length === 0 ? (
        <Empty look="sheet">Every torrent is matched with a Node.</Empty>
      ) : (
        <div className="flex flex-col">
          <p className="mb-6 max-w-[64ch] text-ink-1">
            Re-add checks the files already on disk and downloads only what&apos;s missing.
          </p>
          {groups.map((group) => (
            <GroupSection key={`${group.reason}:${group.found}`} group={group} />
          ))}
        </div>
      )}
    </Page>
  );
}
