package lib3d

import "math"

// ====================================================================
// EBENE  (Port von lib-3d.ts: Plane)
// ====================================================================

// Plane ist eine Ebene im 3D-Raum: n · x + d = 0.
//
// KONVENTION: y zeigt nach oben!
//   - Boden (XZ-Ebene bei y=0): Normal = (0, 1, 0), Distance = 0
//   - Wand (bei x=5):           Normal = (1, 0, 0), Distance = -5
//
// Optional kann die Ebene durch ein konvexes Polygon (Boundary) begrenzt
// werden. Dann prüft IntersectLine automatisch, ob der Schnittpunkt
// innerhalb des Polygons liegt.
type Plane struct {
	Normal   Vec3    // Normalenvektor (muss normiert sein!)
	Distance float64 // d in n·x + d = 0
	Boundary []Vec3  // optionales begrenzendes Polygon (konvex, CCW bzgl. Normal)
}

// NewPlane erzeugt eine Ebene.
func NewPlane(normal Vec3, distance float64, boundary []Vec3) Plane {
	return Plane{Normal: normal, Distance: distance, Boundary: boundary}
}

// IntersectLine berechnet den Schnittpunkt einer Strecke (p1→p2) mit der
// Ebene. Besitzt die Ebene eine Boundary (Polygon), wird zusätzlich geprüft,
// ob der Schnittpunkt innerhalb des Polygons liegt.
//
// Rückgabe: Schnittpunkt und true bei Treffer, sonst false (parallel,
// außerhalb der Strecke oder außerhalb des Polygons).
func (pl Plane) IntersectLine(p1, p2 Vec3) (Vec3, bool) {
	dir := p2.Sub(p1)
	denom := pl.Normal.Dot(dir)

	// Strecke parallel zur Ebene → kein Schnitt
	if math.Abs(denom) < 1e-10 {
		return Vec3{}, false
	}

	t := -(pl.Normal.Dot(p1) + pl.Distance) / denom

	// Schnitt liegt außerhalb der Strecke
	if t < 0 || t > 1 {
		return Vec3{}, false
	}

	hit := p1.Add(dir.Scale(t))

	// Wenn die Ebene begrenzt ist → Polygon-Test
	if len(pl.Boundary) > 0 && !IsPointInConvexPolygon(hit, pl.Boundary, pl.Normal) {
		return Vec3{}, false
	}

	return hit, true
}

// CreatePlaneFromFace erzeugt eine Ebene aus einem Face (3 oder 4 Vertices)
// inklusive Boundary. Die ersten 3 Vertices definieren die Ebene, alle
// Vertices bilden das Polygon.
func CreatePlaneFromFace(faceVerts []Vec3) Plane {
	normal := faceVerts[1].Sub(faceVerts[0]).
		Cross(faceVerts[2].Sub(faceVerts[0])).
		Normalize()
	distance := -normal.Dot(faceVerts[0])
	boundary := make([]Vec3, len(faceVerts))
	copy(boundary, faceVerts)
	return Plane{Normal: normal, Distance: distance, Boundary: boundary}
}

// IsPointInConvexPolygon prüft per Edge-Cross-Test, ob ein Punkt innerhalb
// eines konvexen Polygons liegt.
func IsPointInConvexPolygon(p Vec3, polygon []Vec3, normal Vec3) bool {
	n := len(polygon)
	for i := 0; i < n; i++ {
		a := polygon[i]
		b := polygon[(i+1)%n]
		edge := b.Sub(a)
		toPoint := p.Sub(a)
		if edge.Cross(toPoint).Dot(normal) < 0 {
			return false
		}
	}
	return true
}
