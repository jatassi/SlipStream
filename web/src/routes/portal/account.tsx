import { Link, createFileRoute } from "@tanstack/react-router";

import { accountSearch } from "../../account/search";
import { YourAccount, loadYourAccount } from "../../account/your-account";
import { WhereToWatch, loadMyAccess } from "../../media-servers/access/portal";
import { useAccount } from "../../session/session";
import { fromPortal } from "../../shell/account";
import { Button } from "../../ui/button";
import { Skeleton } from "../../ui/skeleton";

export const Route = createFileRoute("/portal/account")({
  staticData: { title: "Your Account" },
  validateSearch: accountSearch,
  loader: ({ context }) =>
    Promise.all([loadYourAccount(context.queryClient), loadMyAccess(context.queryClient)]),
  pendingComponent: () => <Skeleton label="Loading Your Account" shape="facts" />,
  component: Account,
});

function Account() {
  const { admin } = useAccount();
  const { add } = Route.useSearch();
  const navigate = Route.useNavigate();
  return (
    <YourAccount
      adding={add === "passkey"}
      onAddingChange={(adding) => {
        void navigate({ search: adding ? { add: "passkey" } : {}, replace: true });
      }}
      action={
        admin ? (
          <Button icon={fromPortal.icon} nativeButton={false} render={<Link to={fromPortal.to} />}>
            {fromPortal.label}
          </Button>
        ) : undefined
      }
    >
      {!admin && <WhereToWatch />}
    </YourAccount>
  );
}
