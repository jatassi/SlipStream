import { Button } from "../ui/button";
import { Empty } from "../ui/empty";

/** Nothing is set up for use away from home: one sentence, and the button that shows the suggestions. */
export function NothingWanted({
  suggested,
  choosing,
  onChoose,
}: {
  suggested: boolean;
  choosing: boolean;
  onChoose: () => void;
}) {
  return (
    <div className="mb-6">
      <Empty
        look="sheet"
        action={
          !choosing &&
          suggested && <Button onClick={onChoose}>Choose what to use away from home</Button>
        }
      >
        {suggested
          ? "Nothing is set up for use away from home."
          : "Nothing is set up for use away from home, and nothing is suggested. SlipStream suggests a Media Server once its remote access is on."}
      </Empty>
    </div>
  );
}
