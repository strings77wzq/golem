// Redact scrubs likely secrets from a single line of E2E transcript
// output. It is conservative by design: false positives (over-redaction)
// are far cheaper than leaking a real credential into a CI artefact.
//
// Two redaction layers run in order:
//
//  1. Pattern substitution: known credential shapes (Bearer tokens,
//     OpenAI-style "sk-..." keys, generic "Authorization: ..." headers,
//     "api[_-]?key=..." query/form parameters) are replaced in-place
//     with the literal "[REDACTED]". Surrounding context is preserved
//     so operators can still tell WHICH credential was scrubbed.
//
//  2. Drop heuristic: if a line contains an unbroken alphanumeric run of
//     length ≥ uncertainSecretMinLen and we did NOT recognise the line
//     as a structured-log key/value pair, the entire line is dropped
//     (returned as the empty string). This catches secrets we forgot
//     to add a pattern for; the cost is occasional loss of an
//     innocent base64 blob, which the suite tolerates.
//
// Redact never returns its input verbatim when a secret was matched —
// it always replaces or drops. Callers that need to detect "did
// anything change?" should compare input and output.
package helpers

import (
	"fmt"
	"regexp"
)

// uncertainSecretMinLen is the minimum unbroken alphanumeric run length
// that triggers the conservative drop heuristic. 60 is long enough to
// avoid dropping ordinary identifiers (UUIDs are 36 chars with hyphens,
// JWT segments are typically 30–50 alphanumerics) while still catching
// most leaked secrets in their base64 / hex form.
const uncertainSecretMinLen = 60

// Compiled once at package load so per-line redaction is allocation-light.
var (
	// Bearer tokens: "Bearer <token>" where <token> is ≥ 20 chars from
	// the URL-safe alphabet plus dot/slash/equals (covers JWTs).
	rxBearer = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9_\-\.\/=]{20,}`)

	// OpenAI-style keys: "sk-" or "sk-proj-" followed by ≥ 20 chars.
	rxOpenAIKey = regexp.MustCompile(`sk-(?:proj-)?[A-Za-z0-9_\-]{20,}`)

	// Anthropic-style keys.
	rxAnthropicKey = regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{20,}`)

	// api_key / api-key / apiKey / authorization in URL query or env-style
	// "key=value" pairs.
	rxKVSecret = regexp.MustCompile(`(?i)\b(api[_\-]?key|authorization|x-api-key|access[_\-]?token|secret)\s*[=:]\s*[A-Za-z0-9_\-\.\/=]{12,}`)

	// Long unbroken alphanumeric run for the drop heuristic. Compiled
	// from uncertainSecretMinLen so the constant remains the single
	// source of truth for the heuristic's threshold.
	rxLongRun = regexp.MustCompile(fmt.Sprintf(`[A-Za-z0-9]{%d,}`, uncertainSecretMinLen))

	// Lines we will NOT drop even if they have a long run, because they
	// look like structured Golem log output (event=foo key=bar). Adjust
	// as new prefixes appear in transcripts.
	rxStructuredLog = regexp.MustCompile(`^(level|ts|time|event|msg|tool|model|provider)=`)
)

// Redact returns line with secrets removed. It never returns the same
// pointer as the input; callers may safely retain the result.
func Redact(line string) string {
	// Pattern substitution layer.
	line = rxBearer.ReplaceAllString(line, "[REDACTED]")
	line = rxAnthropicKey.ReplaceAllString(line, "[REDACTED]")
	line = rxOpenAIKey.ReplaceAllString(line, "[REDACTED]")
	line = rxKVSecret.ReplaceAllStringFunc(line, func(match string) string {
		// Preserve the key portion so operators see what was redacted.
		// Split on the first "=" or ":" — whichever appears first.
		for i, ch := range match {
			if ch == '=' || ch == ':' {
				return match[:i+1] + "[REDACTED]"
			}
		}
		return "[REDACTED]"
	})

	// Drop heuristic. Only fires if the long run survived pattern
	// substitution AND the line is not recognised structured log output.
	if rxLongRun.MatchString(line) && !rxStructuredLog.MatchString(line) {
		return ""
	}
	return line
}
