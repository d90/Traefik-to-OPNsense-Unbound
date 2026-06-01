package parser

import "regexp"

var (
	hostRuleRE = regexp.MustCompile(`(?i)Host\(([^)]+)\)`)
	backtickRE = regexp.MustCompile("`([^`]+)`")
)

// ParseHosts extracts hostnames from a Traefik route match rule.
// Handles: Host(`foo.example.com`) and Host(`a.com`, `b.com`)
func ParseHosts(match string) []string {
	ruleMatch := hostRuleRE.FindStringSubmatch(match)
	if len(ruleMatch) < 2 {
		return nil
	}
	tickMatches := backtickRE.FindAllStringSubmatch(ruleMatch[1], -1)
	hosts := make([]string, 0, len(tickMatches))
	for _, m := range tickMatches {
		if len(m) > 1 {
			hosts = append(hosts, m[1])
		}
	}
	return hosts
}
