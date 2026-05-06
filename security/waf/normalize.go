package waf

func Normalize(input []byte) []byte {
	out := make([]byte, 0, len(input))
	pendingSpace := false

	for i := 0; i < len(input); {
		b := input[i]

		if b == '/' && i+1 < len(input) && input[i+1] == '*' {
			i += 2
			for i+1 < len(input) && !(input[i] == '*' && input[i+1] == '/') {
				i++
			}
			if i+1 >= len(input) {
				return trimTrailingSpace(out)
			}
			i += 2
			pendingSpace = true
			continue
		}

		if isWhitespace(b) {
			pendingSpace = true
			i++
			continue
		}

		if pendingSpace && len(out) > 0 {
			out = append(out, ' ')
		}
		pendingSpace = false
		out = append(out, lowerASCII(b))
		i++
	}

	return trimTrailingSpace(out)
}

func isWhitespace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	default:
		return false
	}
}

func trimTrailingSpace(value []byte) []byte {
	if len(value) > 0 && value[len(value)-1] == ' ' {
		return value[:len(value)-1]
	}
	return value
}
