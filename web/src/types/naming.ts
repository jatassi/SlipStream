import type { TokenBreakdown } from './import'

export type ModuleNamingSettings = {
  moduleType: string
  renameEnabled: boolean
  colonReplacement: string
  customColonReplacement: string
  patterns: Record<string, string>
  tokenContexts?: TokenContext[]
  formatOptions?: FormatOptions
}

export type UpdateModuleNamingRequest = {
  renameEnabled?: boolean
  colonReplacement?: string
  customColonReplacement?: string
  multiEpisodeStyle?: string
  patterns?: Record<string, string>
}

export type NamingPreviewRequest = {
  contextName: string
  pattern: string
}

export type NamingPreviewResponse = {
  pattern: string
  preview: string
  valid: boolean
  error?: string
  tokens?: TokenBreakdown[]
}

export type TokenContext = {
  name: string
  label: string
  description: string
  tokens: Token[]
}

export type Token = {
  token: string
  name: string
  description: string
  example: string
}

export type FormatOptions = {
  colonReplacements?: ColonReplacementOption[]
  multiEpisodeStyles?: MultiEpisodeStyleOption[]
}

export type ColonReplacementOption = {
  value: string
  label: string
  example: string
}

export type MultiEpisodeStyleOption = {
  value: string
  label: string
  example: string
}
