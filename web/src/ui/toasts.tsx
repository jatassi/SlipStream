import { Toast } from "@base-ui/react/toast";
import { Cancel01Icon } from "@hugeicons/core-free-icons";

import { Icon } from "./icon";
import { useShell } from "./shell-context";
import { SignalChip } from "./signal-chip";
import { toastManager } from "./toast";

const place = {
  desktop: "top-[72px] right-7 items-end",
  phone: "inset-x-4 top-[calc(env(safe-area-inset-top)+8px)] items-center",
} as const;

export function Toasts() {
  const shell = useShell();
  return (
    <Toast.Provider toastManager={toastManager} limit={3}>
      <Toast.Portal>
        <Toast.Viewport className={`fixed z-50 flex flex-col gap-2 outline-none ${place[shell]}`}>
          <ToastList />
        </Toast.Viewport>
      </Toast.Portal>
    </Toast.Provider>
  );
}

const capsule =
  "flex max-w-[min(440px,calc(100vw-32px))] items-center gap-3 rounded-capsule bg-solid px-4 py-3 text-body font-medium text-on-solid shadow-toast [transform:translate(var(--toast-swipe-movement-x),var(--toast-swipe-movement-y))] transition-[translate,opacity] duration-[220ms] ease-standard data-limited:hidden data-swiping:duration-0 data-starting-style:opacity-0 data-ending-style:opacity-0 motion-safe:data-starting-style:-translate-y-2 motion-safe:data-ending-style:-translate-y-2";

const onSolidButton =
  "shrink-0 rounded-capsule text-on-solid hover:bg-on-solid/10 focus-visible:outline-on-solid";

function ToastList() {
  return Toast.useToastManager().toasts.map((item) => (
    <Toast.Root key={item.id} toast={item} swipeDirection={["up", "right"]} className={capsule}>
      {item.type === "fail" && <SignalChip kind="fail">Failed</SignalChip>}
      <Toast.Title render={<p className="min-w-0 flex-1" />} />
      {item.actionProps && (
        <>
          <Toast.Action
            onClick={() => {
              toastManager.close(item.id);
            }}
            className={`${onSolidButton} h-7 px-3 text-button ring-1 ring-on-solid/20 ring-inset`}
          />
          <Toast.Close
            aria-label="Dismiss"
            className={`${onSolidButton} grid size-7 place-items-center`}
          >
            <Icon icon={Cancel01Icon} className="size-4" />
          </Toast.Close>
        </>
      )}
    </Toast.Root>
  ));
}
