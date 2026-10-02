package lib3d

import "math"

// ====================================================================
// TYPEN  (Port von lib-3d.ts: Vec3, Vec2, Matrix4x4)
// ====================================================================

// Matrix4x4 ist eine 4x4-Matrix in Zeilen-Darstellung (row-major), wie im
// TypeScript-Original number[][]. Für OpenGL wird sie später in ein flaches
// column-major Float32-Array gewandelt (siehe flattenMatrix).
type Matrix4x4 [4][4]float64

// Vec3 ist ein 3D-Vektor.
type Vec3 struct {
	X, Y, Z float64
}

// Vec2 ist ein 2D-Vektor mit Skalierungsfaktor (s) für z. B. Punktgröße.
type Vec2 struct {
	X, Y, S float64
}

// ====================================================================
// VEC3 – ARITHMETIK
// ====================================================================

// Add liefert this + v.
func (a Vec3) Add(v Vec3) Vec3 {
	return Vec3{a.X + v.X, a.Y + v.Y, a.Z + v.Z}
}

// Sub liefert this - v.
func (a Vec3) Sub(v Vec3) Vec3 {
	return Vec3{a.X - v.X, a.Y - v.Y, a.Z - v.Z}
}

// Scale liefert this * s.
func (a Vec3) Scale(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}

// Mag skaliert auf eine bestimmte Ziel-Länge. Der Nullvektor bleibt null.
func (a Vec3) Mag(m float64) Vec3 {
	length := a.Length()
	if length == 0 {
		return Vec3{0, 0, 0}
	}
	return a.Scale(m / length)
}

// Limit begrenzt die Länge auf einen Maximalwert.
func (a Vec3) Limit(max float64) Vec3 {
	mSq := a.SquaredLength()
	if mSq > max*max {
		return a.Scale(max / math.Sqrt(mSq))
	}
	return a.Clone()
}

// Negate kehrt das Vorzeichen um (-this).
func (a Vec3) Negate() Vec3 {
	return Vec3{-a.X, -a.Y, -a.Z}
}

// ====================================================================
// VEC3 – PRODUKTE & BETRAG
// ====================================================================

// Dot liefert das Skalarprodukt this · v.
func (a Vec3) Dot(v Vec3) float64 {
	return a.X*v.X + a.Y*v.Y + a.Z*v.Z
}

// Cross liefert das Kreuzprodukt this × v.
func (a Vec3) Cross(v Vec3) Vec3 {
	return Vec3{
		a.Y*v.Z - a.Z*v.Y,
		a.Z*v.X - a.X*v.Z,
		a.X*v.Y - a.Y*v.X,
	}
}

// SquaredLength liefert das Quadrat der Länge (√-frei).
func (a Vec3) SquaredLength() float64 {
	return a.X*a.X + a.Y*a.Y + a.Z*a.Z
}

// Length liefert die Länge (Betrag) des Vektors.
func (a Vec3) Length() float64 {
	return math.Sqrt(a.SquaredLength())
}

// DistanceTo liefert die Distanz zu einem anderen Vektor.
func (a Vec3) DistanceTo(v Vec3) float64 {
	return a.Sub(v).Length()
}

// ====================================================================
// VEC3 – NORMALISIERUNG & INTERPOLATION
// ====================================================================

// Normalize liefert den Einheitsvektor. Der Nullvektor bleibt null.
func (a Vec3) Normalize() Vec3 {
	length := a.Length()
	if length == 0 {
		return Vec3{0, 0, 0}
	}
	return a.Scale(1 / length)
}

// Lerp ist die lineare Interpolation: this + (v - this) * t
// (t = 0 → this, t = 1 → v).
func (a Vec3) Lerp(v Vec3, t float64) Vec3 {
	return a.Add(v.Sub(a).Scale(t))
}

// ====================================================================
// VEC3 – UTILITY
// ====================================================================

// Clone liefert eine Kopie des Vektors.
func (a Vec3) Clone() Vec3 {
	return Vec3{a.X, a.Y, a.Z}
}

// Equals vergleicht mit Toleranz epsilon (Default im Aufrufer: 1e-10).
func (a Vec3) Equals(v Vec3, epsilon float64) bool {
	return math.Abs(a.X-v.X) < epsilon &&
		math.Abs(a.Y-v.Y) < epsilon &&
		math.Abs(a.Z-v.Z) < epsilon
}

// ====================================================================
// VEC3 – TRANSFORMATION
// ====================================================================

// Transform wendet eine 4x4-Matrix auf den Punkt an (inkl. Translation,
// da die vierte Spalte einbezogen wird).
func (a Vec3) Transform(m Matrix4x4) Vec3 {
	return Vec3{
		m[0][0]*a.X + m[0][1]*a.Y + m[0][2]*a.Z + m[0][3],
		m[1][0]*a.X + m[1][1]*a.Y + m[1][2]*a.Z + m[1][3],
		m[2][0]*a.X + m[2][1]*a.Y + m[2][2]*a.Z + m[2][3],
	}
}

// TransformDir wendet nur den Rotationsteil einer Matrix an (ohne
// Translation) – für Richtungen wie Licht-/Normalenvektoren.
func (a Vec3) TransformDir(m Matrix4x4) Vec3 {
	return Vec3{
		a.X*m[0][0] + a.Y*m[0][1] + a.Z*m[0][2],
		a.X*m[1][0] + a.Y*m[1][1] + a.Z*m[1][2],
		a.X*m[2][0] + a.Y*m[2][1] + a.Z*m[2][2],
	}
}
