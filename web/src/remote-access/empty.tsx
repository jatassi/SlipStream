import { useState } from "react";

import type { ReachabilitySuggestionView } from "../api/gen/model";
import { Button } from "../ui/button";
import { Empty } from "../ui/empty";
import { Suggestions } from "./suggestions";

/** Nothing is set up for use away from home: one sentence, and the suggestions behind its button. */
export function NothingWanted({
  suggestions,
}: {
  suggestions: readonly ReachabilitySuggestionView[];
}) {
  const [choosing, setChoosing] = useState(false);
  return (
    <div className="flex flex-col gap-4">
      <Empty
        look="sheet"
        action={
          !choosing &&
          suggestions.length > 0 && (
            <Button
              onClick={() => {
                setChoosing(true);
              }}
            >
              Choose what to use away from home
            </Button>
          )
        }
      >
        {suggestions.length > 0
          ? "Nothing is set up for use away from home."
          : "Nothing is set up for use away from home. SlipStream suggests the UI and portal once Address is a public https:// name, and each Media Server with remote access on."}
      </Empty>
      {choosing && <Suggestions suggestions={suggestions} />}
    </div>
  );
}
