import { useListGrabsSuspense } from "../api/gen/query/grabs/grabs";
import { Section } from "../ui/section";
import { GrabFailure } from "./failure-cards";
import { GrabTable } from "./grab-table";
import { isFailed, withoutRetriedFailures } from "./stages";

function useWorkGrabs(workId: string) {
  return withoutRetriedFailures(
    useListGrabsSuspense().data.grabs.filter((grab) => grab.workId === workId),
  );
}

export function NeedsYou({ workId, work }: { workId: string; work: string }) {
  const failed = useWorkGrabs(workId).filter((grab) => isFailed(grab));
  if (failed.length === 0) {
    return null;
  }
  return (
    <Section title="Needs you" count={failed.length}>
      <div className="flex flex-col gap-3">
        {failed.map((grab) => (
          <GrabFailure key={grab.id} grab={grab} work={work} form="compact" />
        ))}
      </div>
    </Section>
  );
}

export function Grabs({ workId, work }: { workId: string; work: string }) {
  const grabs = useWorkGrabs(workId).filter((grab) => !isFailed(grab));
  return (
    <Section title="Grabs" count={grabs.length}>
      {grabs.length === 0 ? (
        <p className="text-ink-3">No Grabs yet. A search that finds a release grabs it.</p>
      ) : (
        <GrabTable layout="work" rows={grabs.map((grab) => ({ grab, work }))} />
      )}
    </Section>
  );
}
