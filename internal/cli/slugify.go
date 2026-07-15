package cli

import "strings"

// slugify converts a free-form title into a kebab-case slug candidate: lowercase,
// non-alphanumeric runs collapsed to a single "-", with leading/trailing dashes trimmed.
// The result is only a candidate — callers still gate it through content.IsValidSlug,
// which rejects empty or otherwise malformed output.
func slugify(title string) string {
	var b strings.Builder
	prevDash := false

	for _, r := range strings.ToLower(title) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			// collapse any run of separators/symbols into a single dash
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}
