import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef } from "react";

import {
  useGetWorkSuspense,
  useListContentTypesSuspense,
} from "../../../api/gen/query/works/works";
import { useRemoved } from "../../../live/removed";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";
import { toast } from "../../../ui/toast";
import { typeOf } from "../../../works/fields";
import { loadWork } from "../../../works/page/load";
import { WorkPage } from "../../../works/page/work-page";

export const Route = createFileRoute("/_app/works/$workId")({
  staticData: { title: loaderTitle },
  loader: ({ context, params: { workId } }) => loadWork(context.queryClient, workId),
  pendingComponent: () => <Skeleton label="Loading the Work" />,
  component: WorkRoute,
});

// A Work deleted while its page is open leaves for its Library, saying so
// unless the delete started here, whose own toast already did.
function WorkRoute() {
  const { workId } = Route.useParams();
  const { title, trail } = Route.useLoaderData();
  const removed = useRemoved(workId);
  const deletedHere = useRef(false);
  const navigate = useNavigate();
  useEffect(() => {
    if (!removed) {
      return;
    }
    if (!deletedHere.current) {
      toast(`${title} was deleted.`);
    }
    void navigate({ to: trail.at(-1)?.pathname ?? "/library", replace: true });
  }, [removed, title, trail, navigate]);
  if (removed) {
    return null;
  }
  return (
    <Shown
      workId={workId}
      onDeleting={() => {
        deletedHere.current = true;
      }}
    />
  );
}

function Shown({ workId, onDeleting }: { workId: string; onDeleting: () => void }) {
  const work = useGetWorkSuspense(workId).data;
  const type = typeOf(useListContentTypesSuspense().data.contentTypes, work.contentType);
  return <WorkPage type={type} work={work} onDeleting={onDeleting} />;
}
