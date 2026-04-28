package waf

import "strings"

type Match struct {
	Pattern string
	Field   string
}

type node struct {
	next   map[byte]*node
	fail   *node
	output string
}

type Automaton struct {
	root *node
}

func New(patterns []string) *Automaton {
	root := &node{next: make(map[byte]*node)}
	a := &Automaton{root: root}

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		current := root
		normalized := normalize(pattern)
		for i := 0; i < len(normalized); i++ {
			b := normalized[i]
			if current.next[b] == nil {
				current.next[b] = &node{next: make(map[byte]*node)}
			}
			current = current.next[b]
		}
		current.output = pattern
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
				if child.output == "" {
					child.output = child.fail.output
				}
			}
		}
	}

	return a
}

type Scanner struct {
	automaton *Automaton
	state     *node
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
		b := lowerASCII(raw)

		for s.state != s.automaton.root && s.state.next[b] == nil {
			s.state = s.state.fail
		}

		if next := s.state.next[b]; next != nil {
			s.state = next
		}

		if s.state.output != "" {
			return &Match{
				Pattern: s.state.output,
				Field:   field,
			}
		}
	}

	return nil
}

func normalize(value string) string {
	return strings.ToLower(value)
}

func lowerASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
