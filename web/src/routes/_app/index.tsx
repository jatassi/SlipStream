import { Logout01Icon } from "@hugeicons/core-free-icons";
import { Link, createFileRoute } from "@tanstack/react-router";

import { useDeleteSession } from "../../api/gen/query/identity/identity";
import { useAccount } from "../../session/session";
import { Button } from "../../ui/button";
import { Icon } from "../../ui/icon";

export const Route = createFileRoute("/_app/")({
  component: Home,
});

function Home() {
  const account = useAccount();
  const { endSession } = Route.useRouteContext();
  const signOut = useDeleteSession({
    mutation: {
      onSuccess: () => {
        endSession("signed-out");
      },
    },
  });
  return (
    <section className="flex flex-col items-start gap-4">
      <h1 className="text-2xl font-bold">Signed in as {account.displayName}</h1>
      <Link to="/system" className="underline">
        This Node
      </Link>
      <Button
        onClick={() => {
          signOut.mutate();
        }}
        disabled={signOut.isPending}
      >
        <Icon icon={Logout01Icon} />
        Sign out
      </Button>
    </section>
  );
}
