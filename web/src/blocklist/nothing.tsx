import { Button } from "../ui/button";
import { Empty } from "../ui/empty";

export function Nothing({ q, onClear }: { q: string; onClear: () => void }) {
  if (q !== "") {
    return (
      <Empty look="sheet" action={<Button onClick={onClear}>Clear search</Button>}>
        {`No Blocklist entry has "${q}" in its title.`}
      </Empty>
    );
  }
  return (
    <Empty look="sheet">
      The Blocklist is empty. SlipStream adds a Release when its download fails because of the
      Release itself, such as a tracker reporting it unregistered. You add one when you mark its
      download failed, or when you cancel its Grab and blocklist it.
    </Empty>
  );
}
