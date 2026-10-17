import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { AccountTable } from "../../../accounts/account-table";
import { PendingLinks } from "../../../accounts/pending-links";
import { NewSignupLinkButton, NewSignupLinkDialog } from "../../../accounts/signup-link-dialog";
import {
  getListAccountsSuspenseQueryOptions,
  getListSignupLinksSuspenseQueryOptions,
  useListAccountsSuspense,
  useListSignupLinksSuspense,
} from "../../../api/gen/query/identity/identity";
import { ensureQuery } from "../../../api/query-client";
import { Page } from "../../../shell/page";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/accounts/")({
  staticData: { title: "Accounts" },
  validateSearch: z.object({ newLink: z.literal(true).optional().catch(undefined) }),
  loader: ({ context }) =>
    Promise.all([
      ensureQuery(context.queryClient, getListAccountsSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListSignupLinksSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading Accounts" />,
  component: AccountsPage,
});

function AccountsPage() {
  const { accounts } = useListAccountsSuspense().data;
  const { links } = useListSignupLinksSuspense().data;
  const { newLink } = Route.useSearch();
  const navigate = Route.useNavigate();
  const setLinkOpen = (open: boolean) => {
    void navigate({ search: open ? { newLink: true } : {}, replace: !open });
  };
  return (
    <Page
      action={
        <NewSignupLinkButton
          onClick={() => {
            setLinkOpen(true);
          }}
        />
      }
    >
      <div className="flex flex-col gap-6">
        <AccountTable accounts={accounts} />
        <PendingLinks links={links} />
      </div>
      <NewSignupLinkDialog open={newLink === true} onOpenChange={setLinkOpen} />
    </Page>
  );
}
