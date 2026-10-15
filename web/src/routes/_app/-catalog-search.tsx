import { useNavigate, useSearch } from "@tanstack/react-router";
import { z } from "zod";

import { SearchCatalogQueryParams } from "../../api/gen/zod/works/works.zod";
import { useAppForm } from "../../ui/form";

export const searchForm = z.object({
  query: z.string().trim().pipe(SearchCatalogQueryParams.shape.query),
});

function useSearchForm() {
  const { query } = useSearch({ from: "/_app/add" });
  const navigate = useNavigate({ from: "/add" });
  return useAppForm({
    defaultValues: { query: query ?? "" },
    validators: { onSubmit: searchForm },
    onSubmit: async ({ value }) => {
      await navigate({ search: (search) => ({ ...search, ...searchForm.parse(value) }) });
    },
  });
}

export function CatalogSearch() {
  const form = useSearchForm();
  return (
    <search className="mb-6 block max-w-[640px]">
      <form
        noValidate
        className="flex flex-wrap items-end gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.AppForm>
          <form.AppField name="query">
            {(field) => (
              <div className="min-w-0 flex-[1_1_240px]">
                <field.TextField label="Title" type="search" />
              </div>
            )}
          </form.AppField>
          <form.SubmitButton label="Search" />
        </form.AppForm>
      </form>
    </search>
  );
}
