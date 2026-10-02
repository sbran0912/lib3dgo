package lib3d

import (
	"strconv"
	"strings"
)

// ====================================================================
// FARBE  (Port von lib-wgl.ts: parseColor + ColorState)
// ====================================================================

// ColorState ist eine RGBA-Farbe mit Werten 0..1.
type ColorState struct {
	R, G, B, A float64
}

// ColorArgs normalisiert einen ColorInput (String | Grau | [r,g,b] | [r,g,b,a])
// auf die varargs-Form von FillColor/StrokeColor.
func ColorArgs(c interface{}) []interface{} {
	switch v := c.(type) {
	case []interface{}:
		return v
	case []int:
		out := make([]interface{}, len(v))
		for i, x := range v {
			out[i] = x
		}
		return out
	case []float64:
		out := make([]interface{}, len(v))
		for i, x := range v {
			out[i] = x
		}
		return out
	case nil:
		return nil
	default:
		return []interface{}{c}
	}
}

// ParseColor parst varargs in einen ColorState mit Werten 0..1.
//
// Formen:
//   - 1 String          → Hex ("#rgb" / "#rrggbb")
//   - 1 Zahl            → Graustufe 0..255
//   - 3 Zahlen          → r,g,b 0..255
//   - 4 Zahlen          → r,g,b,a 0..255
func ParseColor(c ...interface{}) ColorState {
	if len(c) == 1 {
		if s, ok := c[0].(string); ok {
			return parseHex(s)
		}
		if v, ok := toFloat(c[0]); ok {
			g := v / 255
			return ColorState{g, g, g, 1}
		}
	}
	if len(c) == 3 {
		r, _ := toFloat(c[0])
		g, _ := toFloat(c[1])
		b, _ := toFloat(c[2])
		return ColorState{r / 255, g / 255, b / 255, 1}
	}
	if len(c) == 4 {
		r, _ := toFloat(c[0])
		g, _ := toFloat(c[1])
		b, _ := toFloat(c[2])
		a, _ := toFloat(c[3])
		return ColorState{r / 255, g / 255, b / 255, a / 255}
	}
	return ColorState{1, 1, 1, 1}
}

// parseHex parst einen Hex-Farbstring in einen ColorState.
func parseHex(s string) ColorState {
	h := strings.TrimPrefix(s, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return ColorState{1, 1, 1, 1}
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return ColorState{1, 1, 1, 1}
	}
	return ColorState{
		R: float64((n>>16)&0xff) / 255,
		G: float64((n>>8)&0xff) / 255,
		B: float64(n&0xff) / 255,
		A: 1,
	}
}

// toFloat wandelt ein interface{} in einen float64 (für Zahlenwerte).
func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// colEq vergleicht zwei Farbzustände exakt.
func colEq(a, b ColorState) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A
}
