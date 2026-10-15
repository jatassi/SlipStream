import { Link } from "@tanstack/react-router";

import type { WorksListedWork } from "../api/gen/model";
import { yearOf } from "./fields";

export function WorkList({ works }: { works: readonly WorksListedWork[] }) {
  if (works.length === 0) {
    return <p className="text-ink-3">Nothing in the library yet.</p>;
  }
  return (
    <ul className="flex flex-col gap-2">
      {works.map((work) => {
        const year = yearOf(work.fields);
        return (
          <li key={work.id}>
            <Link to="/works/$workId" params={{ workId: work.id }} className="underline">
              {work.label}
            </Link>
            {year === undefined ? null : <span className="text-ink-3"> {year}</span>}
          </li>
        );
      })}
    </ul>
  );
}
