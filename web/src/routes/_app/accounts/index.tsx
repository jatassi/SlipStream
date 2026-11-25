import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { AccountTable } from "../../../accounts/account-table";
import { NeedsYou } from "../../../accounts/needs-you";
import { PendingLinks } from "../../../accounts/pending-links";
import { NewSignupLinkButton, NewSignupLinkDialog } from "../../../accounts/signup-link-dialog";
import {
  getListAccountsSuspenseQueryOptions,
  getListSignupLinksSuspenseQueryOptions,
  useListAccountsSuspense,
  useListSignupLinksSuspense,
} from "../../../api/gen/query/identity/identity";
import { getListMediaServerAccessSuspenseQueryOptions } from "../../../api/gen/query/mediaservers/mediaservers";
import { ensureQuery } from "../../../api/query-client";
import { Page, PagePending } from "../../../shell/page";

export const Route = createFileRoute("/_app/accounts/")({
  staticData: { title: "Accounts" },
  validateSearch: z.object({ newLink: z.literal(true).optional().catch(undefined) }),
  loader: ({ context }) =>
    Promise.all([
      ensureQuery(context.queryClient, getListAccountsSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListSignupLinksSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListMediaServerAccessSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <PagePending label="Loading Accounts" shape="table" lines={1} />,
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
        <NeedsYou />
        <AccountTable accounts={accounts} />
        <PendingLinks links={links} />
      </div>
      <NewSignupLinkDialog open={newLink === true} onOpenChange={setLinkOpen} />
    </Page>
  );
}
