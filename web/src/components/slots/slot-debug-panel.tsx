import { useState } from 'react'

import { Bug, ChevronDown, ChevronUp } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'

import { ParseReleaseTester } from './parse-release-tester'
import { ProfileMatchTester } from './profile-match-tester'

export function SlotDebugPanel() {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <Card className="border-dashed border-orange-500/50 bg-orange-500/5">
        <CollapsibleTrigger>
          <CardHeader className="cursor-pointer">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Bug className="size-5 text-orange-500" />
                <CardTitle className="text-orange-500">Debug Tools</CardTitle>
                <Badge variant="outline" className="border-orange-500 text-orange-500">
                  Developer Mode
                </Badge>
              </div>
              {isOpen ? (
                <ChevronUp className="text-muted-foreground size-4" />
              ) : (
                <ChevronDown className="text-muted-foreground size-4" />
              )}
            </div>
            <CardDescription>
              Test release parsing and profile matching without affecting your library
            </CardDescription>
          </CardHeader>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <CardContent className="space-y-6">
            <ParseReleaseTester />
            <ProfileMatchTester />
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  )
}
