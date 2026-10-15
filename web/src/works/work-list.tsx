import { Link } from "@tanstack/react-router";

import type { WorksListedWork } from "../api/gen/model";
import { List, ListRow } from "../ui/list";
import { yearOf } from "./fields";

export function WorkList({ works }: { works: readonly WorksListedWork[] }) {
  return (
    <List label="Works">
      {works.map((work) => (
        <ListRow
          key={work.id}
          title={work.label}
          meta={yearOf(work.fields)}
          render={<Link to="/works/$workId" params={{ workId: work.id }} />}
        />
      ))}
    </List>
  );
}
