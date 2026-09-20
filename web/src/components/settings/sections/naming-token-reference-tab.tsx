import { Group } from '@/components/grouped-list'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'

import { TOKEN_REFERENCE } from './file-naming-constants'

function TokenReferenceGroup() {
  return (
    <Group header="Token Reference" footer="Available tokens for naming patterns.">
      <div className="px-4">
        <Accordion>
          {Object.entries(TOKEN_REFERENCE).map(([category, tokens]) => (
            <AccordionItem key={category} value={category}>
              <AccordionTrigger className="capitalize">
                {category.replaceAll(/([A-Z])/g, ' $1').trim()} Tokens
              </AccordionTrigger>
              <AccordionContent>
                <div className="space-y-2">
                  {tokens.map((t) => (
                    <div
                      key={t.token}
                      className="flex flex-wrap items-start gap-x-4 gap-y-1 border-b py-2 last:border-0"
                    >
                      <code className="bg-muted rounded px-2 py-1 font-mono text-footnote">
                        {t.token}
                      </code>
                      <div className="text-footnote min-w-0 flex-1">
                        <p>{t.description}</p>
                        <p className="text-muted-foreground mt-1">
                          Example: <span className="font-mono">{t.example}</span>
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </Group>
  )
}

function TokenModifiersGroup() {
  return (
    <Group header="Token Modifiers" footer="Additional formatting options for tokens.">
      <div className="text-footnote space-y-4 px-4 py-3">
        <div>
          <h4 className="mb-2 font-medium">Separator Control</h4>
          <p className="text-muted-foreground mb-2">Control word separation within tokens:</p>
          <ul className="text-muted-foreground list-inside list-disc space-y-1">
            <li>
              <code>{'{Series Title}'}</code> - Space separator (default)
            </li>
            <li>
              <code>{'{Series.Title}'}</code> - Period separator
            </li>
            <li>
              <code>{'{Series-Title}'}</code> - Dash separator
            </li>
            <li>
              <code>{'{Series_Title}'}</code> - Underscore separator
            </li>
          </ul>
        </div>

        <div>
          <h4 className="mb-2 font-medium">Truncation</h4>
          <p className="text-muted-foreground mb-2">Limit token length to prevent path issues:</p>
          <ul className="text-muted-foreground list-inside list-disc space-y-1">
            <li>
              <code>{'{Episode Title:30}'}</code> - Truncate to 30 chars from end
            </li>
            <li>
              <code>{'{Episode Title:-30}'}</code> - Truncate to 30 chars from start
            </li>
          </ul>
        </div>
      </div>
    </Group>
  )
}

export function TokenReferenceTab() {
  return (
    <>
      <TokenReferenceGroup />
      <TokenModifiersGroup />
    </>
  )
}
