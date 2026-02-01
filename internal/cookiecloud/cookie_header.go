package cookiecloud

import "strings"

// FindCookiesForDomain returns cookies for the first matching domain key.
// CookieCloud keys are typically cookie.Domain, which may or may not have a leading dot.
func FindCookiesForDomain(cookieData map[string][]Cookie, domain string) []Cookie {
	if len(cookieData) == 0 {
		return nil
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return nil
	}

	for _, k := range candidateDomains(domain) {
		if cookies, ok := cookieData[k]; ok {
			return cookies
		}
	}
	return nil
}

// BuildCookieHeader formats cookies for use as an HTTP "Cookie" header value.
func BuildCookieHeader(cookies []Cookie) string {
	if len(cookies) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		if c.Name == "" {
			continue
		}
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

func candidateDomains(domain string) []string {
	if strings.HasPrefix(domain, ".") {
		trimmed := strings.TrimPrefix(domain, ".")
		return []string{domain, trimmed}
	}
	return []string{domain, "." + domain}
}
