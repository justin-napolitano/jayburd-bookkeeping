import type {
  OnboardingCandidate,
  OnboardingCatalog,
  OnboardingCommand,
  OnboardingPending,
  OnboardingPreview,
} from "./onboarding";
// Explicitly fictional metadata shared by unit and component browser proofs only.
export const onboardingIDs = {
  workspace: "81000000-0000-4000-8000-000000000001",
  candidate: "81000000-0000-4000-8000-000000000002",
  entity: "81000000-0000-4000-8000-000000000003",
  book: "81000000-0000-4000-8000-000000000004",
  source: "81000000-0000-4000-8000-000000000005",
  preview: "81000000-0000-4000-8000-000000000006",
  command: "81000000-0000-4000-8000-000000000007",
  recovery: "81000000-0000-4000-8000-000000000008",
};
export const onboardingCandidate: OnboardingCandidate = {
  candidate_id: onboardingIDs.candidate,
  entity_id: onboardingIDs.entity,
  book_id: onboardingIDs.book,
  entity_name: "Fictional Studio",
  book_name: "Fictional records",
  snapshot: "a".repeat(64),
  sources: [
    { source_id: onboardingIDs.source, display_name: "Fictional checking" },
  ],
};
export const onboardingCatalog = (ready = false): OnboardingCatalog => ({
  profile: "private-onboarding-catalog-v1",
  workspace: ready
    ? {
        state: "ready",
        workspace_id: onboardingIDs.workspace,
        authority_revision: "1",
        capabilities: ["read", "acquire", "classify"],
      }
    : { state: "absent" },
  can_setup: !ready,
  can_adopt: ready,
  candidates: ready ? [onboardingCandidate] : [],
});
export const onboardingPreview = (adopt = false): OnboardingPreview =>
  ({
    profile: "private-onboarding-preview-v1",
    preview_id: onboardingIDs.preview,
    expires_at: new Date(Date.now() + 900000)
      .toISOString()
      .replace(/\.\d{3}Z$/, "Z"),
    snapshot: "a".repeat(64),
    ...(adopt
      ? {
          operation: "adopt",
          workspace_id: onboardingIDs.workspace,
          review: {
            ...onboardingCandidate,
            expected_generation: "1",
            impact: "pause_legacy_acquisition",
          },
        }
      : {
          operation: "setup",
          workspace_id: null,
          review: { capabilities: ["read", "acquire", "classify"] },
        }),
  }) as OnboardingPreview;
export const onboardingCommand = (
  preview = onboardingPreview(),
  prepare_key = "fictional-prepare",
): OnboardingCommand => ({
  profile: "private-onboarding-command-v1",
  command_id: onboardingIDs.command,
  recovery_id: onboardingIDs.recovery,
  prepare_key,
  state: "prepared",
  preview,
  result: null,
  acknowledged: false,
});
export function commitOnboarding(
  command: OnboardingCommand,
): OnboardingCommand {
  return {
    ...command,
    state: "committed",
    result:
      command.preview.operation === "setup"
        ? {
            workspace_id: onboardingIDs.workspace,
            workspace_revision: "1",
            capabilities: ["read", "acquire", "classify"],
          }
        : {
            collection_item_id: onboardingIDs.candidate,
            item_generation: "2",
            sources: [
              {
                source_account_id: onboardingIDs.source,
                source_revision: "0",
                revision: "1",
              },
            ],
          },
  };
}
export const onboardingPending = (
  command: OnboardingCommand | null = null,
): OnboardingPending => ({
  profile: "private-onboarding-pending-v1",
  state: command ? "available" : "none",
  command,
  recovery_id: command ? command.recovery_id : null,
});
