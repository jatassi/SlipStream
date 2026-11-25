import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";

import {
  getGetAccountSuspenseQueryOptions,
  useGetAccountSuspense,
} from "../../../api/gen/query/identity/identity";
import { getListMediaServerAccessSuspenseQueryOptions } from "../../../api/gen/query/mediaservers/mediaservers";
import { ensureQuery } from "../../../api/query-client";
import { Page, PagePending } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { AccountSections } from "./-account-sections";
import { AccountDialogs, type Asking } from "./-confirmations";

export const Route = createFileRoute("/_app/accounts/$accountId")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params: { accountId } }) => {
    const [{ account }] = await Promise.all([
      ensureQuery(context.queryClient, getGetAccountSuspenseQueryOptions(accountId)),
      ensureQuery(
        context.queryClient,
        getListMediaServerAccessSuspenseQueryOptions({ account: accountId }),
      ),
    ]);
    return { title: account.displayName, trail: [{ title: "Accounts", pathname: "/accounts" }] };
  },
  pendingComponent: () => <PagePending label="Loading the Account" shape="facts" />,
  component: AccountPage,
});

function AccountPage() {
  const { accountId } = Route.useParams();
  const detail = useGetAccountSuspense(accountId).data;
  const navigate = Route.useNavigate();
  const [asking, setAsking] = useState<Asking>();
  const close = () => {
    setAsking(undefined);
  };
  return (
    <Page>
      <AccountSections detail={detail} onAsk={setAsking} />
      <AccountDialogs
        account={detail.account}
        asking={asking}
        onClose={close}
        onDeleted={() => navigate({ to: "/accounts" })}
      />
    </Page>
  );
}
