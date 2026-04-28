package waf

var defaultPatterns = []string{
	"union select",
	"<script",
	"javascript:",
	"' or 1=1",
	"../",
}

func DefaultPatterns() []string {
	patterns := make([]string, len(defaultPatterns))
	copy(patterns, defaultPatterns)
	return patterns
}

func NewDefaultAutomaton() *Automaton {
	return New(DefaultPatterns())
}
