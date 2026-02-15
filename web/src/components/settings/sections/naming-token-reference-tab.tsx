import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

import { TOKEN_REFERENCE } from './file-naming-constants'

function TokenReferenceCard() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Token Reference</CardTitle>
        <CardDescription>Available tokens for naming patterns</CardDescription>
      </CardHeader>
      <CardContent>
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
                      className="flex items-start gap-4 border-b py-2 last:border-0"
                    >
                      <code className="bg-muted min-w-[180px] rounded px-2 py-1 font-mono text-sm">
                        {t.token}
                      </code>
                      <div className="flex-1 text-sm">
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
      </CardContent>
    </Card>
  )
}

function TokenModifiersCard() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Token Modifiers</CardTitle>
        <CardDescription>Additional formatting options for tokens</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
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
          <p className="text-muted-foreground mb-2">
            Limit token length to prevent path issues:
          </p>
          <ul className="text-muted-foreground list-inside list-disc space-y-1">
            <li>
              <code>{'{Episode Title:30}'}</code> - Truncate to 30 chars from end
            </li>
            <li>
              <code>{'{Episode Title:-30}'}</code> - Truncate to 30 chars from start
            </li>
          </ul>
        </div>
      </CardContent>
    </Card>
  )
}

export function TokenReferenceTab() {
  return (
    <>
      <TokenReferenceCard />
      <TokenModifiersCard />
    </>
  )
}
