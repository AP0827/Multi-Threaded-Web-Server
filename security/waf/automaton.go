package waf

type Match struct {
	Pattern string
	Field   string
	RuleID  string
}

type node struct {
	next   map[byte]*node
	fail   *node
	output *Rule
}

type Automaton struct {
	root *node
}

type Rule struct {
	ID      string
	Pattern string
}

func New(patterns []string) *Automaton {
	rules := make([]Rule, 0, len(patterns))
	for i, pattern := range patterns {
		if pattern == "" {
			continue
		}
		rules = append(rules, Rule{
			ID:      ruleID(i + 1),
			Pattern: pattern,
		})
	}
	return NewRules(rules)
}

func NewRules(rules []Rule) *Automaton {
	root := &node{next: make(map[byte]*node)}
	a := &Automaton{root: root}

	for i := range rules {
		rule := rules[i]
		if rule.Pattern == "" {
			continue
		}
		if rule.ID == "" {
			rule.ID = ruleID(i + 1)
		}

		current := root
		normalized := Normalize([]byte(rule.Pattern))
		for i := 0; i < len(normalized); i++ {
			b := normalized[i]
			if current.next[b] == nil {
				current.next[b] = &node{next: make(map[byte]*node)}
			}
			current = current.next[b]
		}
		current.output = &rule
	}

	queue := make([]*node, 0, len(root.next))
	for _, child := range root.next {
		child.fail = root
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for b, child := range current.next {
			queue = append(queue, child)

			fail := current.fail
			for fail != nil && fail.next[b] == nil {
				fail = fail.fail
			}

			if fail == nil {
				child.fail = root
			} else {
				child.fail = fail.next[b]
				if child.output == nil {
					child.output = child.fail.output
				}
			}
		}
	}

	return a
}

type Scanner struct {
	automaton          *Automaton
	state              *node
	pendingSpace       bool
	pendingSlash       bool
	inBlockComment     bool
	pendingCommentStar bool
	emitted            bool
}

func (a *Automaton) NewScanner() *Scanner {
	if a == nil || a.root == nil {
		return &Scanner{}
	}

	return &Scanner{
		automaton: a,
		state:     a.root,
	}
}

func (s *Scanner) Feed(field string, data []byte) *Match {
	if s == nil || s.automaton == nil || s.automaton.root == nil {
		return nil
	}

	for _, raw := range data {
		if match := s.feedRaw(field, raw); match != nil {
			return match
		}
	}

	return nil
}

func (s *Scanner) Finish(field string) *Match {
	if s == nil || s.automaton == nil || s.automaton.root == nil {
		return nil
	}
	if s.pendingSlash {
		s.pendingSlash = false
		if s.pendingSpace && s.emitted {
			if match := s.feedNormalized(field, ' '); match != nil {
				return match
			}
		}
		s.pendingSpace = false
		if match := s.feedNormalized(field, '/'); match != nil {
			return match
		}
	}
	s.pendingSpace = false
	s.inBlockComment = false
	s.pendingCommentStar = false
	return nil
}

func (s *Scanner) feedRaw(field string, raw byte) *Match {
	if s.inBlockComment {
		if s.pendingCommentStar && raw == '/' {
			s.inBlockComment = false
			s.pendingCommentStar = false
			s.pendingSpace = true
			return nil
		}
		s.pendingCommentStar = raw == '*'
		return nil
	}

	if s.pendingSlash {
		s.pendingSlash = false
		if raw == '*' {
			s.inBlockComment = true
			s.pendingSpace = true
			return nil
		}
		if s.pendingSpace && s.emitted {
			if match := s.feedNormalized(field, ' '); match != nil {
				return match
			}
		}
		s.pendingSpace = false
		if match := s.feedNormalized(field, '/'); match != nil {
			return match
		}
	}

	if raw == '/' {
		s.pendingSlash = true
		return nil
	}
	if isWhitespace(raw) {
		s.pendingSpace = true
		return nil
	}

	if s.pendingSpace && s.emitted {
		if match := s.feedNormalized(field, ' '); match != nil {
			return match
		}
	}
	s.pendingSpace = false

	return s.feedNormalized(field, lowerASCII(raw))
}

func (s *Scanner) feedNormalized(field string, b byte) *Match {
	s.emitted = true
	for s.state != s.automaton.root && s.state.next[b] == nil {
		s.state = s.state.fail
	}

	if next := s.state.next[b]; next != nil {
		s.state = next
	}

	if s.state.output != nil {
		return &Match{
			Pattern: s.state.output.Pattern,
			RuleID:  s.state.output.ID,
			Field:   field,
		}
	}

	return nil
}

func ruleID(index int) string {
	return "MTWS-WAF-" + zeroPad(index, 4)
}

func zeroPad(value int, width int) string {
	text := ""
	for value > 0 {
		text = string(byte('0'+value%10)) + text
		value /= 10
	}
	if text == "" {
		text = "0"
	}
	for len(text) < width {
		text = "0" + text
	}
	return text
}

func lowerASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
