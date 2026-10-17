import { useNavigate } from "@tanstack/react-router";

import { Button } from "../../ui/button";

export function OpenSlipStream({ sentence, to }: { sentence: string; to: "/" | "/portal" }) {
  const navigate = useNavigate();
  return (
    <>
      <p className="text-ink-1">{sentence}</p>
      <Button
        look="primary"
        onClick={() => {
          void navigate({ to, replace: true });
        }}
      >
        Open SlipStream
      </Button>
    </>
  );
}
