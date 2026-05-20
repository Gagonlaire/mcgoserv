package commander

import "strings"

const (
	usageOptionalOpen  = "["
	usageOptionalClose = "]"
	usageRequiredOpen  = "("
	usageRequiredClose = ")"
	usageOr            = "|"
	usageSeparator     = " "
)

func (d *Dispatcher) UsageLines(node *Node, pathPrefix string, perm int) []string {
	selfExecutable := node.Run != nil
	var lines []string
	for _, child := range node.Children {
		if child.Redirect != nil {
			continue
		}
		usage, ok := smartUsage(child, perm, selfExecutable, selfExecutable)
		if !ok {
			continue
		}
		if pathPrefix == "" {
			lines = append(lines, "/"+usage)
		} else {
			lines = append(lines, "/"+pathPrefix+usageSeparator+usage)
		}
	}
	return lines
}

func smartUsage(node *Node, perm int, optional, deep bool) (string, bool) {
	if node.PermissionLevel > 0 && node.PermissionLevel > perm {
		return "", false
	}

	self := node.UsageText()
	if optional {
		self = usageOptionalOpen + self + usageOptionalClose
	}
	if deep {
		return self, true
	}

	if node.Redirect != nil {
		return self, true
	}

	childOptional := node.Run != nil
	usable := make([]*Node, 0, len(node.Children))
	for _, c := range node.Children {
		if c.PermissionLevel > 0 && c.PermissionLevel > perm {
			continue
		}
		usable = append(usable, c)
	}

	switch len(usable) {
	case 0:
		return self, true
	case 1:
		sub, ok := smartUsage(usable[0], perm, childOptional, childOptional)
		if !ok {
			return self, true
		}
		return self + usageSeparator + sub, true
	default:
		seen := map[string]struct{}{}
		var distinct []string
		for _, c := range usable {
			sub, ok := smartUsage(c, perm, childOptional, true)
			if !ok {
				continue
			}
			if _, dup := seen[sub]; !dup {
				seen[sub] = struct{}{}
				distinct = append(distinct, sub)
			}
		}
		switch len(distinct) {
		case 0:
			return self, true
		case 1:
			u := distinct[0]
			if childOptional {
				u = usageOptionalOpen + u + usageOptionalClose
			}
			return self + usageSeparator + u, true
		default:
			open, close := usageRequiredOpen, usageRequiredClose
			if childOptional {
				open, close = usageOptionalOpen, usageOptionalClose
			}
			var tokens []string
			for _, c := range usable {
				tokens = append(tokens, c.UsageText())
			}
			return self + usageSeparator + open + strings.Join(tokens, usageOr) + close, true
		}
	}
}
