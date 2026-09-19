import type { useVersionSlotsSection } from '@/components/settings/sections/use-version-slots-section'
import { DryRunModal, ResolveConfigModal, ResolveNamingModal } from '@/components/slots'

type SectionState = ReturnType<typeof useVersionSlotsSection>

export function SlotMigrationModals({ section }: { section: SectionState }) {
  return (
    <>
      <ResolveConfigModal
        open={section.resolveConfigOpen}
        onOpenChange={section.setResolveConfigOpen}
        conflicts={section.validationResult?.conflicts ?? []}
        onResolved={() => void section.handleValidate()}
      />
      <ResolveNamingModal
        open={section.resolveNamingOpen}
        onOpenChange={section.setResolveNamingOpen}
        missingMovieTokens={section.namingValidation?.movieValidation.missingTokens}
        missingEpisodeTokens={section.namingValidation?.episodeValidation.missingTokens}
        onResolved={() => void section.handleValidateNaming()}
      />
      <DryRunModal
        open={section.dryRunOpen}
        onOpenChange={section.setDryRunOpen}
        onMigrationComplete={section.handleMigrationComplete}
        onMigrationFailed={(error) => section.setMigrationError(error)}
      />
    </>
  )
}
