export function normalizeReviewText(value: string): string {
  return value.toLowerCase().replace(/[\s\u00a0]+/g, '').replace(/[“”‘’]/g, '"')
}

export function hasReviewQuoteMatch(renderedText: string, quote: string): boolean {
  const haystack = normalizeReviewText(renderedText)
  const needle = normalizeReviewText(quote)
  if (!needle) return false
  if (haystack.includes(needle)) return true
  if (needle.length < 40) return false
  const samples = [needle.slice(0, 40), needle.slice(Math.max(0, Math.floor(needle.length / 2) - 20), Math.floor(needle.length / 2) + 20), needle.slice(-40)]
  return samples.filter((sample) => haystack.includes(sample)).length >= 2
}
