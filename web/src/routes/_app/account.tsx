import { createFileRoute } from "@tanstack/react-router";

import { accountSearch } from "../../account/search";
import { YourAccount, loadYourAccount } from "../../account/your-account";
import { Skeleton } from "../../ui/skeleton";

export const Route = createFileRoute("/_app/account")({
  staticData: { title: "Your Account" },
  validateSearch: accountSearch,
  loader: ({ context }) => loadYourAccount(context.queryClient),
  pendingComponent: () => <Skeleton label="Loading Your Account" />,
  component: Account,
});

function Account() {
  const { add } = Route.useSearch();
  const navigate = Route.useNavigate();
  return (
    <YourAccount
      adding={add === "passkey"}
      onAddingChange={(adding) => {
        void navigate({ search: adding ? { add: "passkey" } : {}, replace: true });
      }}
    />
  );
}
