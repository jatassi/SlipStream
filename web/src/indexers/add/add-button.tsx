import { Add01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import { Button } from "../../ui/button";

export function AddIndexerButton() {
  return (
    <Button nativeButton={false} icon={Add01Icon} render={<Link to="/settings/indexers/new" />}>
      Add Indexer
    </Button>
  );
}
