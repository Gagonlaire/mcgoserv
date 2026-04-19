package nbtpath

import (
	"fmt"
	"strconv"
	"strings"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Tnze/go-mc/nbt"
)

const (
	colorKey    = tc.ColorAqua
	colorNum    = tc.ColorGold
	colorType   = tc.ColorRed
	colorString = tc.ColorGreen
	colorPunct  = tc.ColorWhite
)

func FormatSNBTComponent(v any) *tc.TextComponent {
	root := tc.Container()
	appendSNBT(root, v)
	return root
}

func SNBTToComponent(snbt nbt.StringifiedMessage) (*tc.TextComponent, error) {
	value, err := SNBTToValue(snbt)
	if err != nil {
		return nil, err
	}
	return FormatSNBTComponent(value), nil
}

func appendSNBT(c *tc.TextComponent, v any) {
	switch val := v.(type) {
	case map[string]any:
		c.AddExtra(punct("{"))
		first := true
		for k, vv := range val {
			if !first {
				c.AddExtra(punct(", "))
			}
			first = false
			appendKey(c, k)
			c.AddExtra(punct(": "))
			appendSNBT(c, vv)
		}
		c.AddExtra(punct("}"))

	case []any:
		c.AddExtra(punct("["))
		for i, elem := range val {
			if i > 0 {
				c.AddExtra(punct(", "))
			}
			appendSNBT(c, elem)
		}
		c.AddExtra(punct("]"))

	case []byte:
		appendArrayHeader(c, "B")
		for i, b := range val {
			if i > 0 {
				c.AddExtra(punct(", "))
			}
			appendNumber(c, strconv.FormatInt(int64(int8(b)), 10), "b")
		}
		c.AddExtra(punct("]"))

	case []int32:
		appendArrayHeader(c, "I")
		for i, n := range val {
			if i > 0 {
				c.AddExtra(punct(", "))
			}
			c.AddExtra(tc.Text(strconv.FormatInt(int64(n), 10)).SetColor(colorNum))
		}
		c.AddExtra(punct("]"))

	case []int64:
		appendArrayHeader(c, "L")
		for i, n := range val {
			if i > 0 {
				c.AddExtra(punct(", "))
			}
			appendNumber(c, strconv.FormatInt(n, 10), "l")
		}
		c.AddExtra(punct("]"))

	case int8:
		appendNumber(c, strconv.FormatInt(int64(val), 10), "b")
	case int16:
		appendNumber(c, strconv.FormatInt(int64(val), 10), "s")
	case int32:
		c.AddExtra(tc.Text(strconv.FormatInt(int64(val), 10)).SetColor(colorNum))
	case int64:
		appendNumber(c, strconv.FormatInt(val, 10), "l")
	case float32:
		appendNumber(c, formatFloat(float64(val), 32), "f")
	case float64:
		appendNumber(c, formatFloat(val, 64), "d")
	case string:
		appendString(c, val)
	case bool:
		if val {
			appendNumber(c, "1", "b")
		} else {
			appendNumber(c, "0", "b")
		}
	default:
		c.AddExtra(tc.Text(fmt.Sprintf("%v", val)).SetColor(colorPunct))
	}
}

func punct(s string) *tc.TextComponent {
	return tc.Text(s).SetColor(colorPunct)
}

func appendArrayHeader(c *tc.TextComponent, typeLetter string) {
	c.AddExtra(punct("["))
	c.AddExtra(tc.Text(typeLetter).SetColor(colorType))
	c.AddExtra(punct("; "))
}

func appendNumber(c *tc.TextComponent, num, suffix string) {
	c.AddExtra(tc.Text(num).SetColor(colorNum))
	c.AddExtra(tc.Text(suffix).SetColor(colorType))
}

func appendKey(c *tc.TextComponent, key string) {
	safe := key != ""
	for _, b := range []byte(key) {
		if !isUnquotedChar(b) {
			safe = false
			break
		}
	}
	if safe {
		c.AddExtra(tc.Text(key).SetColor(colorKey))
		return
	}
	appendString(c, key)
}

func formatFloat(v float64, bits int) string {
	s := strconv.FormatFloat(v, 'g', -1, bits)
	if !strings.ContainsAny(s, ".eEnN") {
		s += ".0"
	}
	return s
}

func appendString(c *tc.TextComponent, s string) {
	var inner strings.Builder
	for _, r := range s {
		if r == '"' || r == '\\' {
			inner.WriteByte('\\')
		}
		inner.WriteRune(r)
	}
	c.AddExtra(punct(`"`))
	c.AddExtra(tc.Text(inner.String()).SetColor(colorString))
	c.AddExtra(punct(`"`))
}
