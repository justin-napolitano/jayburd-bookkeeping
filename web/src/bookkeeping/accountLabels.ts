// Current-view labels distinguish private account identities without exposing them as headings.
// Sorting makes a fixed account set independent of provider/list ordering; these are not permanent bank names.
export function accountLabels(ids: string[]): (id: string) => string {
  const labels = new Map(
    [...new Set(ids)]
      .sort()
      .map((id, index) => [id, `Bank account ${index + 1}`]),
  );
  return (id) => labels.get(id) ?? "Bank account unavailable";
}
