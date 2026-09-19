import type { ConsoleMessage, Page } from '@playwright/test'

function isErrorMessage(message: ConsoleMessage): boolean {
  return message.type() === 'error'
}

export function captureConsoleErrors(page: Page): string[] {
  const errors: string[] = []
  page.on('pageerror', (error) => {
    errors.push(error.message)
  })
  page.on('console', (message) => {
    if (isErrorMessage(message)) {
      errors.push(message.text())
    }
  })
  return errors
}
