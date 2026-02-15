import { toast } from 'sonner'

export function withToast<T extends unknown[]>(
  fn: (...args: T) => Promise<void>,
  errorMsg: string,
) {
  return async (...args: T) => {
    try {
      await fn(...args)
    } catch {
      toast.error(errorMsg)
    }
  }
}
