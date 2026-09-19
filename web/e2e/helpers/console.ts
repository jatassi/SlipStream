import type { ConsoleMessage, Page } from '@playwright/test'

function isErrorMessage(message: ConsoleMessage): boolean {
  return message.type() === 'error'
}

function isIgnoredConsoleError(text: string): boolean {
  if (text.includes('WebSocket connection to') && text.includes('/ws')) {
    return true
  }
  if (text.includes('cannot be a descendant') || text.includes('cannot contain a nested')) {
    return true
  }
  return text.includes('Failed to load resource')
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

export function unexpectedConsoleErrors(errors: string[]): string[] {
  return errors.filter((text) => !isIgnoredConsoleError(text))
}
