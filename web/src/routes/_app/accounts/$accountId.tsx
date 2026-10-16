import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";

import {
  getGetAccountSuspenseQueryOptions,
  useGetAccountSuspense,
} from "../../../api/gen/query/identity/identity";
import { ensureQuery } from "../../../api/query-client";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";
import { AccountSections } from "./-account-sections";
import { AccountDialogs, type Asking } from "./-confirmations";

export const Route = createFileRoute("/_app/accounts/$accountId")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params: { accountId } }) => {
    const { account } = await ensureQuery(
      context.queryClient,
      getGetAccountSuspenseQueryOptions(accountId),
    );
    return { title: account.displayName, trail: [{ title: "Accounts", pathname: "/accounts" }] };
  },
  pendingComponent: () => <Skeleton label="Loading the Account" />,
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
