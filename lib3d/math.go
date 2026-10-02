package lib3d

import (
	"math"
	"math/rand/v2"
)

// ====================================================================
// HILFS-MATHEMATIK  (Port von lib-3d.ts)
// ====================================================================

// Project projiziert einen 3D-Punkt (im Kamerakoordinatensystem) auf den
// Bildschirm. Für das 3D-Rendering übernimmt die GPU die Projektion via
// PerspectiveMatrix; Project ist für 2D-Overlays gedacht.
func Project(fov float64, v Vec3) Vec2 {
	s := fov / (fov + v.Z)
	return Vec2{
		X: v.X * s,
		Y: v.Y * s,
		S: s,
	}
}

// Constrain begrenzt value auf den Bereich [min, max].
func Constrain(value, min, max float64) float64 {
	return math.Min(max, math.Max(min, value))
}

// Random liefert eine ganzzahlige Zufallszahl in [n1, n2).
func Random(n1, n2 int) int {
	return int(math.Floor(rand.Float64()*float64(n2-n1) + float64(n1)))
}

// RandomFloat liefert eine Gleitkomma-Zufallszahl in [min, max).
func RandomFloat(min, max float64) float64 {
	return rand.Float64()*(max-min) + min
}
