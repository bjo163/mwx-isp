// Maps the prose emitted by internal/adminapi/noc_impact.go to i18n keys so the
// NOC impact panel fully follows the active locale; unknown future messages
// fall through untranslated instead of breaking.
type TranslateFn = (key: string, options?: Record<string, unknown>) => string;

const exactMessages: Record<string, string> = {
  'The target address does not match any NAS record, so no subscriber mapping can be derived.':
    'pages.noc.impact.diagnostic.no_nas_record',
  'Subscribers are inferred from RADIUS sessions and accounting observed on this NAS address.':
    'pages.noc.impact.diagnostic.nas_activity_inferred',
  'No subscriber activity was observed on this NAS within the lookback window, so impact cannot be confirmed.':
    'pages.noc.impact.diagnostic.nas_activity_none',
  'Customers are mapped by an explicit ODP assignment.':
    'pages.noc.impact.diagnostic.odp_explicit_assignment',
  'No customer, subscription, or session references this ODP.':
    'pages.noc.impact.diagnostic.odp_no_references',
  'No customer stores this ODP ID; customers are matched by the cached ODP code instead.':
    'pages.noc.impact.diagnostic.odp_cached_code_match',
  'Subscriber impact is unknown for this target: no NAS or ODP association exists.':
    'pages.noc.impact.diagnostic.nas_unknown_mapping',
  'ODP enclosures have no direct probe; this view maps exposure but does not confirm an outage. Online status comes from live RADIUS sessions.':
    'pages.noc.impact.diagnostic.odp_no_probe',
  'No subscribers are assigned to this ODP by ID or cached code; the impact list is empty, not confirmed clear.':
    'pages.noc.impact.diagnostic.odp_unassigned_cached',
  'Impact was inferred from the cached ODP code because customers do not store the ODP ID.':
    'pages.noc.impact.diagnostic.odp_cached_inference',
  'No subscribers are assigned to this ODP; the impact list is empty, not confirmed clear.':
    'pages.noc.impact.diagnostic.odp_unassigned',
  'No monitor target is configured for this NAS address, so freshness and outage state are unknown.':
    'pages.noc.impact.diagnostic.nas_no_monitor',
  'Monitoring is disabled for this target; the current outage state is unknown.':
    'pages.noc.impact.diagnostic.monitoring_disabled',
  'This target has never been checked; the outage state is unknown.':
    'pages.noc.impact.diagnostic.never_checked',
};

const patternMessages: Array<{
  pattern: RegExp;
  key: string;
  params: (match: RegExpMatchArray) => Record<string, string>;
}> = [
  {
    pattern: /^No RADIUS sessions or accounting records were found for this NAS in the last (\d+) hours\.$/,
    key: 'pages.noc.impact.diagnostic.nas_no_radius_records',
    params: (match) => ({ hours: match[1] }),
  },
  {
    pattern:
      /^The last probe failed at (.+), older than the (\d+)-second freshness window\. Treat this as last-known state, not a confirmed live outage\.$/,
    key: 'pages.noc.impact.diagnostic.probe_failed_stale',
    params: (match) => ({ checked_at: match[1], seconds: match[2] }),
  },
  {
    pattern:
      /^The last probe completed at (.+), older than the (\d+)-second freshness window; the outage state cannot be confirmed\.$/,
    key: 'pages.noc.impact.diagnostic.probe_completed_stale',
    params: (match) => ({ checked_at: match[1], seconds: match[2] }),
  },
];

export const translateNocImpactDiagnostic = (translate: TranslateFn, message: string): string => {
  const exactKey = exactMessages[message];
  if (exactKey) return translate(exactKey, { _: message });
  for (const entry of patternMessages) {
    const match = message.match(entry.pattern);
    if (match) return translate(entry.key, { _: message, ...entry.params(match) });
  }
  return message;
};
