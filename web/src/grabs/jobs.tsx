import { useListGrabsSuspense } from "../api/gen/query/grabs/grabs";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { useLibrary } from "../library/projection";
import { Empty } from "../shell/empty";
import { RunningList } from "../tasks/running";
import { Button } from "../ui/button";
import { Select } from "../ui/select";
import { useShell } from "../ui/shell-context";
import { ActivityBody } from "./activity";

const every = "";

type Choice = {
  readonly type: string | undefined;
  readonly onTypeChange: (type: string | undefined) => void;
};

function TypeFilter({
  types,
  type,
  onTypeChange,
}: Choice & { types: readonly { id: string; name: string }[] }) {
  return (
    <Select
      label="Content Type"
      options={[
        { value: every, label: "Every Content Type" },
        ...types.map(({ id, name }) => ({ value: id, label: name })),
      ]}
      value={type ?? every}
      onValueChange={(value) => {
        onTypeChange(value === every ? undefined : value);
      }}
    />
  );
}

/** Every Job, narrowed to one Content Type when more than one has Jobs. */
export function Jobs({ type, onTypeChange }: Choice) {
  const { grabs } = useListGrabsSuspense().data;
  const works = useLibrary();
  const labels = new Map(works.map((work) => [work.id, work.label]));
  const typeOf = new Map(works.map((work) => [work.id, work.type.id]));
  const held = new Set(grabs.map((grab) => typeOf.get(grab.workId)));
  const types = useListContentTypesSuspense().data.contentTypes.filter(
    ({ id }) => held.has(id) || id === type,
  );
  const shown =
    type === undefined ? grabs : grabs.filter((grab) => typeOf.get(grab.workId) === type);
  return (
    <>
      {useShell() === "phone" && <RunningList />}
      {shown.length === 0 && grabs.length > 0 ? (
        <Empty
          action={
            <Button
              onClick={() => {
                onTypeChange(undefined);
              }}
            >
              Show every Content Type
            </Button>
          }
        >
          No jobs for {types.find(({ id }) => id === type)?.name ?? "this Content Type"}.
        </Empty>
      ) : (
        <ActivityBody
          grabs={shown}
          labels={labels}
          filter={
            (types.length > 1 || type !== undefined) && (
              <TypeFilter types={types} type={type} onTypeChange={onTypeChange} />
            )
          }
        />
      )}
    </>
  );
}
