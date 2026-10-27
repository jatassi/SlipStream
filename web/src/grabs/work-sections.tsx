import { useListGrabsSuspense } from "../api/gen/query/grabs/grabs";
import { Section } from "../ui/section";
import { GrabTable } from "./grab-table";
import { isFailed, withoutRetriedFailures } from "./stages";

function useWorkGrabs(workId: string) {
  return withoutRetriedFailures(
    useListGrabsSuspense().data.grabs.filter((grab) => grab.workId === workId),
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
