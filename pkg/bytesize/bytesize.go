package bytesize

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var re = regexp.MustCompile(`([0-9.]+)([kKmMgGtTpP]?)[Bb]?`)

// ErrParse message
var ErrParse = errors.New("parse error")

// Parse kiro mega giga tera peta
func Parse(s string) (uint64, error) {
	p := re.FindStringSubmatch(s)
	if len(p) < 3 {
		return 0, ErrParse
	}
	n, _ := strconv.ParseFloat(p[1], 64)
	switch strings.ToLower(p[2]) {
	case "":
		return uint64(n), nil
	case "k":
		return uint64(n * (1 << 10)), nil
	case "m":
		return uint64(n * (1 << 20)), nil
	case "g":
		return uint64(n * (1 << 30)), nil
	case "t":
		return uint64(n * (1 << 40)), nil
	case "p":
		return uint64(n * (1 << 50)), nil
	}
	return 0, ErrParse
}

// MustParse kiro mega giga tera peta
func MustParse(s string) uint64 {
	i, err := Parse(s)
	if err != nil {
		return 0
	}
	return i
}
