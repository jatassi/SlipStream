export function withToast<T extends unknown[]>(fn: (...args: T) => Promise<void>) {
  return async (...args: T) => {
    await fn(...args)
  }
}
