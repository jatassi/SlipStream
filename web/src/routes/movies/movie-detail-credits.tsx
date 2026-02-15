import { User } from 'lucide-react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { Credits, Person } from '@/types'

type MovieDetailCreditsProps = {
  credits?: Credits
}

export function MovieDetailCredits({ credits }: MovieDetailCreditsProps) {
  if (!credits) {
    return null
  }

  const hasCast = credits.cast.length > 0
  const crewMembers = [
    ...(credits.directors ?? []),
    ...(credits.writers ?? []),
  ]
  const hasCrew = crewMembers.length > 0

  if (!hasCast && !hasCrew) {
    return null
  }

  return (
    <>
      {hasCast ? (
        <Card>
          <CardHeader>
            <CardTitle>Cast</CardTitle>
          </CardHeader>
          <CardContent>
            <PersonList people={credits.cast} max={18} />
          </CardContent>
        </Card>
      ) : null}
      {hasCrew ? (
        <Card>
          <CardHeader>
            <CardTitle>Crew</CardTitle>
          </CardHeader>
          <CardContent>
            <PersonList people={crewMembers} max={12} />
          </CardContent>
        </Card>
      ) : null}
    </>
  )
}

function PersonList({ people, max = 12 }: { people: Person[]; max?: number }) {
  return (
    <div className="flex gap-4 overflow-x-auto pb-2">
      {people.slice(0, max).map((person) => (
        <div
          key={`${person.id}-${person.role}`}
          className="flex w-20 shrink-0 flex-col items-center gap-1"
        >
          <div className="bg-muted flex size-16 items-center justify-center overflow-hidden rounded-full">
            {person.photoUrl ? (
              <img src={person.photoUrl} alt={person.name} className="size-full object-cover" />
            ) : (
              <User className="text-muted-foreground size-8" />
            )}
          </div>
          <span className="line-clamp-2 w-full text-center text-xs">{person.name}</span>
          {person.role ? (
            <span className="text-muted-foreground line-clamp-2 w-full text-center text-xs">
              {person.role}
            </span>
          ) : null}
        </div>
      ))}
    </div>
  )
}
