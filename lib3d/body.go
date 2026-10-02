package lib3d

// ====================================================================
// BODY – Physik-fähiger 3D-Körper  (Port von lib-body.ts)
//
// Kapselt ein Solid mit Position, Rotation, Geschwindigkeit und Farbe.
//
// GPU-Modell: Das Solid hält nur CPU-Geometrie (Vertices/Kanten/Faces in
// flachen Float-Arrays). Der Upload in den GPU-Batch passiert pro Frame in
// draw.go DrawBody(). Es gibt keinen persistenten GPU-Buffer pro Solid und
// kein Retain/Release/Dispose – ein Body kann einfach aus einer Liste
// entfernt werden.
// ====================================================================

// BodyConfig entspricht der C-BodyConfig. Nullwerte bedeuten „nicht gesetzt“
// und werden durch die Defaults ersetzt (Color "#ffffff", LineWidth 1, rot 0).
type BodyConfig struct {
	Color     string
	LineWidth float64
	RotX      float64
	RotY      float64
	RotZ      float64
}

// BodyConfigDefault sind die Standardwerte (entspricht BODY_CONFIG_DEFAULT).
var BodyConfigDefault = BodyConfig{
	Color:     "#ffffff",
	LineWidth: 1,
	RotX:      0,
	RotY:      0,
	RotZ:      0,
}

// Body ist ein 3D-Körper.
type Body struct {
	// Solid ist die Geometrie (CPU – kann zwischen Bodies geteilt werden).
	Solid *Solid

	// Pos ist die Position im Weltraum.
	Pos Vec3

	// Vel ist die Geschwindigkeit (für Physik).
	Vel Vec3

	// RotX/RotY/RotZ ist die Rotation um die Achsen (in Radian).
	RotX, RotY, RotZ float64

	// Darstellung
	Color     string
	LineWidth float64

	// Faces ist die Face-Topologie: Arrays von Vertex-Indizes. Nur für
	// geschlossene Körper (Box, Pyramide, …) gesetzt. nil schaltet die
	// Kollision für diesen Body ab (z. B. für Kugeln mit degenerierten
	// Pol-Quads).
	Faces [][]int
}

// NewBody erzeugt einen Body. cfg darf nil sein (dann gelten die Defaults);
// Nullwerte einzelner Felder werden ebenfalls durch die Defaults ersetzt.
func NewBody(solid *Solid, x, y, z float64, cfg *BodyConfig) *Body {
	c := BodyConfigDefault
	if cfg != nil {
		if cfg.Color != "" {
			c.Color = cfg.Color
		}
		if cfg.LineWidth != 0 {
			c.LineWidth = cfg.LineWidth
		}
		c.RotX = cfg.RotX
		c.RotY = cfg.RotY
		c.RotZ = cfg.RotZ
	}

	return &Body{
		Solid: solid,
		// Single Source of Truth: Kollisions-Topologie = Solid-Topologie.
		Faces:     solid.Faces,
		Pos:       Vec3{x, y, z},
		Vel:       Vec3{0, 0, 0},
		Color:     c.Color,
		LineWidth: c.LineWidth,
		RotX:      c.RotX,
		RotY:      c.RotY,
		RotZ:      c.RotZ,
	}
}

// ================================================================
// FACE / INTERSECTION
// ================================================================

// GetFacePlanes liefert die Face-Planes dieses Körpers in Weltkoordinaten.
// Berücksichtigt Translation und Rotation. Nur Bodies mit gesetzten Faces
// liefern Ergebnisse.
func (b *Body) GetFacePlanes() []Plane {
	if b.Faces == nil || len(b.Faces) == 0 {
		return nil
	}

	m := b.ModelMatrix()
	worldVerts := make([]Vec3, len(b.Solid.Vertices))
	for i, v := range b.Solid.Vertices {
		worldVerts[i] = v.Transform(m)
	}

	planes := make([]Plane, 0, len(b.Faces))
	for _, faceIdx := range b.Faces {
		face := make([]Vec3, len(faceIdx))
		for i, idx := range faceIdx {
			face[i] = worldVerts[idx]
		}
		planes = append(planes, CreatePlaneFromFace(face))
	}
	return planes
}

// ModelMatrix liefert die Modellmatrix (Translation × Rotation) – dieselbe
// Matrix, die das Rendering (draw.go DrawBody) und die Kollision
// (GetFacePlanes) verwenden.
func (b *Body) ModelMatrix() Matrix4x4 {
	t := TranslateMatrix(b.Pos.X, b.Pos.Y, b.Pos.Z)
	if b.RotX == 0 && b.RotY == 0 && b.RotZ == 0 {
		return t
	}
	return MultMatrix(t, RotateMatrix(b.RotX, b.RotY, b.RotZ))
}

// DistanceTo liefert die Distanz (Mittelpunkt zu Mittelpunkt) zu einem
// anderen Body.
func (b *Body) DistanceTo(other *Body) float64 {
	return b.Pos.DistanceTo(other.Pos)
}

// ====================================================================
// FACTORIES
// ====================================================================

// CreateBox erzeugt einen achsenparallelen Quader mit faces-Topologie.
func CreateBox(w, h, d, x, y, z float64, cfg *BodyConfig) *Body {
	return NewBody(CreateBoxSolid(w, h, d), x, y, z, cfg)
}

// CreatePyramid erzeugt eine quadratische Pyramide mit faces-Topologie.
func CreatePyramid(base, height, x, y, z float64, cfg *BodyConfig) *Body {
	return NewBody(CreatePyramidSolid(base, height), x, y, z, cfg)
}

// CreateGrid erzeugt ein Gitter (Grid) in der XZ-Ebene.
func CreateGrid(size float64, cells int, x, y, z float64, cfg *BodyConfig) *Body {
	return NewBody(CreateGridSolid(size, cells), x, y, z, cfg)
}

// CreateSphere erzeugt eine Drahtgitter-Kugel (UV-Sphere).
func CreateSphere(radius float64, slices, stacks int, x, y, z float64, cfg *BodyConfig) *Body {
	sphere := NewBody(CreateSphereSolid(radius, slices, stacks), x, y, z, cfg)
	// Das Solid trägt Faces (gefülltes Rendering), aber die Pol-Quads sind
	// degeneriert → Kollision bewusst abschalten.
	sphere.Faces = nil
	return sphere
}
