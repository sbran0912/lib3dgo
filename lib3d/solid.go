package lib3d

import (
	"math"
	"strconv"
	"strings"
)

// ====================================================================
// SOLID – Ein 3D-Drahtgitter-Objekt (Port von lib-solids.ts)
//
// Trennung der Zuständigkeiten:
//   vec/mat4/plane/math → Mathematik
//   solid.go            → 3D-Objekte (reine CPU-Geometrie)
//   body.go             → Physik-fähige Körper (Position/Rotation/Farbe)
//   renderer.go         → Renderer (Batch-Sammlung, Shader, Primitives)
//   draw.go             → Zeichnen von Solid/Body (Transformation → Batch)
//
// Geschlossene Körper (Box, Pyramide, Kugel) tragen zusätzlich Faces
// (konvexe Polygon-Indizes) und werden dadurch GEFÜLLT und mit Flat Shading
// beleuchtet gerendert – Kanten bleiben als Drahtgitter darüber sichtbar.
//
// GPU-Modell: Ein Solid ist reine CPU-Geometrie. Die Kanten/Flächen werden
// einmal in flache Float-Arrays expandiert (FlatEdges/FlatFaces); der Upload
// in den GPU-Batch passiert pro Frame in draw.go DrawSolid() – es gibt KEINEN
// persistenten GPU-Buffer pro Solid.
// ====================================================================

// Solid ist ein 3D-Objekt als CPU-Geometrie.
type Solid struct {
	// Vertices sind 3D-Punkte in Objekt-Koordinaten (lokal).
	Vertices []Vec3

	// Edges sind Kanten als Paare von Vertex-Indizes.
	Edges [][2]int

	// Faces ist die Face-Topologie: jedes Face ist ein konvexes Polygon aus
	// Vertex-Indizes (3+ Ecken, CCW von außen). Fehlt bei Drahtgittern (Grid).
	Faces [][]int

	// FlatEdges sind die Kanten in flachem Float-Format (zwei Vertices pro
	// Kante, x,y,z,…). Upload in den GPU-Batch pro Frame.
	FlatEdges []float32

	// FlatFaces sind die triangulierten Flächen im flachen Float-Format.
	// Für Solids ohne Face-Daten (z. B. Grid) bleibt FlatFaces leer.
	FlatFaces []float32
}

// NewSolid erzeugt ein Solid und expandiert Kanten/Flächen einmalig.
func NewSolid(vertices []Vec3, edges [][2]int, faces [][]int) *Solid {
	s := &Solid{
		Vertices: vertices,
		Edges:    edges,
		Faces:    faces,
	}

	// Kanten einmalig expandieren.
	edgeVerts := make([]float32, 0, len(edges)*6)
	for _, e := range edges {
		a := s.Vertices[e[0]]
		b := s.Vertices[e[1]]
		edgeVerts = append(edgeVerts, float32(a.X), float32(a.Y), float32(a.Z), float32(b.X), float32(b.Y), float32(b.Z))
	}
	s.FlatEdges = edgeVerts

	// Flächen triangulieren (Fan aus Index 0) und expandieren.
	faceVerts := make([]float32, 0)
	if s.Faces != nil {
		for _, face := range s.Faces {
			if len(face) < 3 {
				continue
			}
			for j := 1; j < len(face)-1; j++ {
				for _, k := range []int{0, j, j + 1} {
					v := s.Vertices[face[k]]
					faceVerts = append(faceVerts, float32(v.X), float32(v.Y), float32(v.Z))
				}
			}
		}
	}
	s.FlatFaces = faceVerts

	return s
}

// ====================================================================
// HILFE – Hex-Farbe abdunkeln
// ====================================================================

// DarkenHex gibt einen um amount (0..1) abgedunkelten Hex-Farbstring zurück.
// amount=0 → unverändert, amount=1 → schwarz.
func DarkenHex(hex string, amount float64) string {
	h := strings.TrimPrefix(hex, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	r, _ := strconv.ParseInt(h[0:2], 16, 32)
	g, _ := strconv.ParseInt(h[2:4], 16, 32)
	b, _ := strconv.ParseInt(h[4:6], 16, 32)
	f := 1 - amount
	return "#" + hex2(int(float64(r)*f)) + hex2(int(float64(g)*f)) + hex2(int(float64(b)*f))
}

// hex2 formatiert einen Wert 0..255 als zweistellige Hex-Zahl.
func hex2(v int) string {
	s := strconv.FormatInt(int64(v), 16)
	if len(s) < 2 {
		return "0" + s
	}
	return s
}

// ====================================================================
// HILFSKONSTRUKTOREN – Standard-Geometrien
// ====================================================================

// CreateBoxSolid erzeugt einen achsenparallelen Quader (Box) mit Zentrum im
// Ursprung: 8 Ecken, 12 Kanten, 6 Flächen.
func CreateBoxSolid(w, h, d float64) *Solid {
	hw, hh, hd := w/2, h/2, d/2
	V := func(x, y, z float64) Vec3 { return Vec3{x, y, z} }

	vertices := []Vec3{
		V(-hw, -hh, -hd), // 0: vorne-unten-links
		V(hw, -hh, -hd),  // 1: vorne-unten-rechts
		V(hw, hh, -hd),   // 2: vorne-oben-rechts
		V(-hw, hh, -hd),  // 3: vorne-oben-links
		V(-hw, -hh, hd),  // 4: hinten-unten-links
		V(hw, -hh, hd),   // 5: hinten-unten-rechts
		V(hw, hh, hd),    // 6: hinten-oben-rechts
		V(-hw, hh, hd),   // 7: hinten-oben-links
	}

	edges := [][2]int{
		// Vorderseite (Z = -hd)
		{0, 1}, {1, 2}, {2, 3}, {3, 0},
		// Rückseite (Z = +hd)
		{4, 5}, {5, 6}, {6, 7}, {7, 4},
		// Verbindungen vorne ↔ hinten
		{0, 4}, {1, 5}, {2, 6}, {3, 7},
	}

	// Flächen: je ein konvexes Quad (CCW von außen).
	faces := [][]int{
		{0, 3, 2, 1}, // vorne
		{4, 5, 6, 7}, // hinten
		{0, 4, 7, 3}, // links
		{1, 2, 6, 5}, // rechts
		{0, 1, 5, 4}, // unten
		{3, 7, 6, 2}, // oben
	}

	return NewSolid(vertices, edges, faces)
}

// CreatePyramidSolid erzeugt eine quadratische Pyramide mit Zentrum im
// Ursprung: 5 Ecken, 8 Kanten, 5 Flächen.
func CreatePyramidSolid(base, height float64) *Solid {
	hb := base / 2
	V := func(x, y, z float64) Vec3 { return Vec3{x, y, z} }

	vertices := []Vec3{
		V(-hb, -height/2, -hb), // 0: Basis-vorne-links
		V(hb, -height/2, -hb),  // 1: Basis-vorne-rechts
		V(hb, -height/2, hb),   // 2: Basis-hinten-rechts
		V(-hb, -height/2, hb),  // 3: Basis-hinten-links
		V(0, height/2, 0),      // 4: Spitze
	}

	edges := [][2]int{
		// Basis
		{0, 1}, {1, 2}, {2, 3}, {3, 0},
		// Seitenkanten
		{0, 4}, {1, 4}, {2, 4}, {3, 4},
	}

	faces := [][]int{
		{0, 1, 4},    // vorne
		{1, 2, 4},    // rechts
		{2, 3, 4},    // hinten
		{3, 0, 4},    // links
		{3, 2, 1, 0}, // Basis
	}

	return NewSolid(vertices, edges, faces)
}

// CreateGridSolid erzeugt ein Gitter (Grid) in der XZ-Ebene.
func CreateGridSolid(size float64, cells int) *Solid {
	half := size / 2
	step := size / float64(cells)

	vertices := make([]Vec3, 0, (cells+1)*(cells+1))
	for iz := 0; iz <= cells; iz++ {
		for ix := 0; ix <= cells; ix++ {
			vertices = append(vertices, Vec3{-half + float64(ix)*step, 0, -half + float64(iz)*step})
		}
	}

	edges := make([][2]int, 0)
	stride := cells + 1

	// Horizontale Linien (entlang X)
	for iz := 0; iz <= cells; iz++ {
		for ix := 0; ix < cells; ix++ {
			idx := iz*stride + ix
			edges = append(edges, [2]int{idx, idx + 1})
		}
	}

	// Vertikale Linien (entlang Z)
	for ix := 0; ix <= cells; ix++ {
		for iz := 0; iz < cells; iz++ {
			idx := iz*stride + ix
			edges = append(edges, [2]int{idx, idx + stride})
		}
	}

	return NewSolid(vertices, edges, nil)
}

// CreateSphereSolid erzeugt eine Drahtgitter-Kugel (UV-Sphere).
func CreateSphereSolid(radius float64, slices, stacks int) *Solid {
	V := func(x, y, z float64) Vec3 { return Vec3{x, y, z} }

	vertices := make([]Vec3, 0)
	edges := make([][2]int, 0)

	// --- Vertices generieren ---
	for i := 0; i <= stacks; i++ {
		theta := (float64(i) / float64(stacks)) * math.Pi
		y := radius * math.Cos(theta)
		r := radius * math.Sin(theta)
		for j := 0; j <= slices; j++ {
			phi := (float64(j) / float64(slices)) * math.Pi * 2
			vertices = append(vertices, V(r*math.Cos(phi), y, r*math.Sin(phi)))
		}
	}

	// --- Kanten: Meridiane (vertikal, Pol zu Pol) ---
	for j := 0; j <= slices; j++ {
		for i := 0; i < stacks; i++ {
			a := i*(slices+1) + j
			b := (i+1)*(slices+1) + j
			edges = append(edges, [2]int{a, b})
		}
	}

	// --- Kanten: Breitenringe (horizontal, Ring für Ring) ---
	for i := 0; i <= stacks; i++ {
		for j := 0; j < slices; j++ {
			a := i*(slices+1) + j
			b := i*(slices+1) + j + 1
			edges = append(edges, [2]int{a, b})
		}
	}

	// --- Flächen: jedes Gitterzellen-Quad (stacks × slices) ist ein Face ---
	faces := make([][]int, 0)
	for i := 0; i < stacks; i++ {
		for j := 0; j < slices; j++ {
			a := i*(slices+1) + j
			b := i*(slices+1) + j + 1
			c := (i+1)*(slices+1) + j + 1
			d := (i+1)*(slices+1) + j
			faces = append(faces, []int{a, b, c, d})
		}
	}

	return NewSolid(vertices, edges, faces)
}
