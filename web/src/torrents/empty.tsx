import { Link } from "@tanstack/react-router";

import { Empty } from "../shell/empty";
import { Button } from "../ui/button";

export function NoTorrents() {
  return (
    <Empty
      action={
        <Button nativeButton={false} render={<Link to="/library" />}>
          Open the Library
        </Button>
      }
    >
      No torrents yet. Open a Work and choose Search to grab a release. Its torrent shows here until
      you remove it.
    </Empty>
  );
}

export function NoMatch({ onClear }: { onClear: (() => void) | undefined }) {
  return (
    <Empty action={onClear && <Button onClick={onClear}>Clear filters</Button>}>
      {onClear === undefined ? "No torrents here right now." : "No torrents match these filters."}
    </Empty>
  );
}
