import { Link, createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import type { WorksCandidate, WorksContentType } from "../../api/gen/model";
import {
  getListContentTypesSuspenseQueryOptions,
  getSearchCatalogSuspenseQueryOptions,
  useAddWork,
  useSearchCatalogSuspense,
} from "../../api/gen/query/works/works";
import { SearchCatalogQueryParams } from "../../api/gen/zod/works/works.zod";
import { ensureQuery } from "../../api/query-client";
import { Button } from "../../ui/button";
import { submitErrors, useAppForm } from "../../ui/form";
import { Skeleton } from "../../ui/skeleton";
import { enabledType, fieldText } from "../../works/fields";

const searchForm = z.object({
  query: z.string().trim().pipe(SearchCatalogQueryParams.shape.query),
});

export const Route = createFileRoute("/_app/add")({
  validateSearch: z.object({ query: searchForm.shape.query.optional().catch(undefined) }),
  loaderDeps: ({ search: { query } }) => ({ query }),
  loader: async ({ context, deps: { query } }) => {
    const { contentTypes } = await ensureQuery(
      context.queryClient,
      getListContentTypesSuspenseQueryOptions(),
    );
    const type = enabledType(contentTypes);
    if (type !== undefined && query !== undefined) {
      await ensureQuery(
        context.queryClient,
        getSearchCatalogSuspenseQueryOptions(type.id, { query }),
      );
    }
    return type;
  },
  pendingComponent: () => <Skeleton label="Loading the catalog" />,
  component: AddPage,
});

function useSearchForm() {
  const { query } = Route.useSearch();
  const navigate = Route.useNavigate();
  return useAppForm({
    defaultValues: { query: query ?? "" },
    validators: { onSubmit: searchForm },
    onSubmit: async ({ value }) => {
      await navigate({ search: searchForm.parse(value) });
    },
  });
}

function AddPage() {
  const type = Route.useLoaderData();
  return type === undefined ? (
    <p>Turn on a Content Type to add to the library.</p>
  ) : (
    <Catalog type={type} />
  );
}

function Catalog({ type }: { type: WorksContentType }) {
  const { query } = Route.useSearch();
  const form = useSearchForm();
  return (
    <section className="flex flex-col gap-6">
      <h1 className="text-2xl font-bold">Add {type.work.one.toLowerCase()}</h1>
      <search>
        <form
          noValidate
          className="flex flex-wrap items-end gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit();
          }}
        >
          <form.AppForm>
            <form.AppField name="query">
              {(field) => <field.TextField label="Title" type="search" />}
            </form.AppField>
            <form.SubmitButton label="Search" />
          </form.AppForm>
        </form>
      </search>
      {query === undefined ? null : <Candidates type={type} query={query} />}
    </section>
  );
}

function Candidates({ type, query }: { type: WorksContentType; query: string }) {
  const { candidates } = useSearchCatalogSuspense(type.id, { query }).data;
  if (candidates.length === 0) {
    return <p className="text-ink-3">Nothing matches “{query}”.</p>;
  }
  return (
    <ul aria-label="Results" className="flex flex-col gap-3">
      {candidates.map((candidate) => (
        <li key={candidate.key} className="flex flex-wrap items-center gap-4">
          <span>
            {candidate.label}
            <span className="text-ink-3"> {candidate.fields.map(fieldText).join(" · ")}</span>
          </span>
          <CandidateAction type={type} candidate={candidate} />
        </li>
      ))}
    </ul>
  );
}

function CandidateAction({
  type,
  candidate,
}: {
  type: WorksContentType;
  candidate: WorksCandidate;
}) {
  const navigate = Route.useNavigate();
  const addWork = useAddWork({
    mutation: {
      onSuccess: async ({ work }) => {
        await navigate({ to: "/works/$workId", params: { workId: work.id } });
      },
    },
  });
  if (candidate.workId !== null) {
    return (
      <Link
        to="/works/$workId"
        params={{ workId: candidate.workId }}
        aria-label={`Open ${candidate.label}`}
        className="underline"
      >
        Open
      </Link>
    );
  }
  return (
    <>
      <Button
        aria-label={`Add ${candidate.label}`}
        disabled={addWork.isPending}
        onClick={() => {
          addWork.mutate({ data: { contentType: type.id, key: candidate.key, monitor: true } });
        }}
      >
        Add
      </Button>
      {addWork.isError ? (
        <p role="alert" className="text-fail-ink">
          {submitErrors(addWork.error).form}
        </p>
      ) : null}
    </>
  );
}
