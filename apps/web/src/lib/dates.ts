// entryDate is a calendar day ("2026-11-03") with no time zone. new Date("2026-11-03") reads it
// as midnight UTC, which shows as 2 Nov anywhere west of UTC. Formatting in UTC keeps the day
// the user picked.
export function formatEntryDate(entryDate: string, locale?: string): string {
  const date = new Date(`${entryDate}T00:00:00Z`)
  if (Number.isNaN(date.getTime())) return entryDate // unexpected format: show it as-is
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeZone: 'UTC' }).format(date)
}