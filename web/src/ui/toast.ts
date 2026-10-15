import { Toast } from "@base-ui/react/toast";

export const toastManager = Toast.createToastManager();

export type ToastKind = "done" | "fail";

export type ToastOptions = {
  action?: { label: string; onClick: () => void };
  kind?: ToastKind;
};

export function toast(message: string, { action, kind = "done" }: ToastOptions = {}) {
  toastManager.add({
    title: message,
    type: kind,
    timeout: action ? 0 : 2800,
    ...(action && { actionProps: { children: action.label, onClick: action.onClick } }),
  });
}
