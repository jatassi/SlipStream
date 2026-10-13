import { Link, createFileRoute } from "@tanstack/react-router";

import {
  getListContentTypesSuspenseQueryOptions,
  getListWorksSuspenseQueryOptions,
  useListContentTypesSuspense,
  useListWorksSuspense,
} from "../../api/gen/query/works/works";
import { ensureQuery } from "../../api/query-client";
import { Skeleton } from "../../ui/skeleton";
import { enabledType, yearOf } from "../../works/fields";

export const Route = createFileRoute("/_app/")({
  loader: ({ context }) =>
    Promise.all([
      ensureQuery(context.queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListWorksSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading the Library" />,
  component: Library,
});

function Library() {
  const type = enabledType(useListContentTypesSuspense().data.contentTypes);
  const { works } = useListWorksSuspense().data;
  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-wrap items-baseline justify-between gap-4">
        <h1 className="text-2xl font-bold">Library</h1>
        {type === undefined ? null : (
          <Link to="/add" className="underline">
            Add {type.work.one.toLowerCase()}
          </Link>
        )}
      </div>
      {works.length === 0 ? (
        <p className="text-ink-3">Nothing in the library yet.</p>
      ) : (
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
      )}
    </section>
  );
}
