package lib3d

// ====================================================================
// ZEICHEN-FASSADE (Port von lib-render.ts)
//
// Solid und Body bleiben reine Daten (Geometrie bzw. Physik + Darstellung),
// das Zeichnen liegt hier. Dadurch müssen solid.go und body.go kein
// OpenGL/GLFW kennen und es gibt keinen Import-Zyklus.
//
// Rollenverteilung:
//   draw.go     → Fassade für das Zeichnen. Die Szene ruft ausschließlich
//                 Draw*() auf; Farbe/Linienbreite/Punktgröße kommen als
//                 Options-Objekt (PrimitiveStyle) mit.
//   renderer.go → Low-Level-Renderer (Batching, Shader, Primitive, Setup).
//
// GPU-Modell: Das Solid hält nur CPU-Geometrie (FlatEdges/FlatFaces). Der
// Upload in den GPU-Batch passiert pro Frame hier.
// ====================================================================

// ColorInput ist eine Farbeingabe für die Fassade:
// Hex-String | Grau | [r,g,b] | [r,g,b,a]. In Go entspricht das interface{}.
type ColorInput interface{}

// PrimitiveStyle sind Zeichen-Optionen. Nullwerte (nil/0) überschreiben den
// aktuellen Low-Level-Zustand nicht.
type PrimitiveStyle struct {
	Stroke    ColorInput // Linienfarbe (unbeleuchtet)
	Fill      ColorInput // Füllfarbe  (beleuchtet)
	LineWidth float64    // Linienstärke in Pixeln
	PointSize float64    // Punktgröße in Pixeln
}

// ShapeStyle erweitert PrimitiveStyle um den Zeichenmodus (und optionale
// Parameter für Rect/Circle).
type ShapeStyle struct {
	PrimitiveStyle
	Mode     DrawStyle // 0 = stroke | 1 = fill | 2 = beide (Standard: 0)
	Z        float64   // optionale Z-Höhe (für DrawRect)
	Segments int       // optionale Segmentzahl (für DrawCircle)
}

// applyStyle übernimmt die gesetzten Felder in den Low-Level-Zeichenzustand.
// Nicht gesetzte Felder bleiben unverändert (wie im Immediate-Mode üblich).
func applyStyle(o PrimitiveStyle) {
	if o.Stroke != nil {
		StrokeColor(ColorArgs(o.Stroke)...)
	}
	if o.Fill != nil {
		FillColor(ColorArgs(o.Fill)...)
	}
	if o.LineWidth != 0 {
		StrokeWidth(o.LineWidth)
	}
	if o.PointSize != 0 {
		PointSize(o.PointSize)
	}
}

// ====================================================================
// SOLIDS / BODIES
// ====================================================================

// DrawSolid zeichnet ein Solid. Pipeline: Objekt-Koordinaten
//
//	→ SetModel(world)   (View ist global pro Frame gesetzt)
//	→ Faces als TRIANGLES in den Batch (Fill-Farbe, beleuchtet)
//	→ Kanten als LINES in den Batch   (Stroke-Farbe, unbelichtet)
//
// Danach wird das Model auf Identität zurückgesetzt.
func DrawSolid(s *Solid, world Matrix4x4) {
	SetModel(world)
	if len(s.FlatFaces) > 0 {
		SubmitTriangles(s.FlatFaces) // Flächen (Fill-Farbe, beleuchtet)
	}
	SubmitLines(s.FlatEdges) // Kanten (Stroke-Farbe, unbelichtet)
	SetModel(IdentityMatrix())
}

// DrawBody zeichnet einen Body. Setzt Farbe und Linienbreite, baut die
// Modellmatrix und zeichnet dessen Solid.
func DrawBody(b *Body) {
	StrokeWidth(b.LineWidth)
	StrokeColor(b.Color)
	FillColor(b.Color) // Füllfarbe für die (beleuchteten) Flächen
	DrawSolid(b.Solid, b.ModelMatrix())
}

// ====================================================================
// IMMEDIATE-PRIMITIVES
// ====================================================================

// DrawPoint zeichnet einen 3D-Punkt bei (x,y,z).
func DrawPoint(x, y, z float64, style PrimitiveStyle) {
	applyStyle(style)
	Point(x, y, z)
}

// DrawLine zeichnet eine 3D-Linie von (x1,y1,z1) nach (x2,y2,z2).
func DrawLine(x1, y1, z1, x2, y2, z2 float64, style PrimitiveStyle) {
	applyStyle(style)
	Line(x1, y1, z1, x2, y2, z2)
}

// DrawTriangle zeichnet ein 3D-Dreieck (Mode: 0=stroke | 1=fill | 2=beide).
func DrawTriangle(x1, y1, z1, x2, y2, z2, x3, y3, z3 float64, style ShapeStyle) {
	applyStyle(style.PrimitiveStyle)
	Triangle(x1, y1, z1, x2, y2, z2, x3, y3, z3, style.Mode)
}

// DrawShape zeichnet ein 3D-Viereck mit 4 beliebigen Punkten.
func DrawShape(x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4 float64, style ShapeStyle) {
	applyStyle(style.PrimitiveStyle)
	Shape(x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4, style.Mode)
}

// DrawRect zeichnet ein achsenparalleles Rechteck in der XY-Ebene.
func DrawRect(x, y, w, h float64, style ShapeStyle) {
	applyStyle(style.PrimitiveStyle)
	Rect(x, y, w, h, style.Mode, style.Z)
}

// DrawCircle zeichnet einen 3D-Kreis (Ring) in der XY-Ebene.
func DrawCircle(x, y, z, radius float64, style ShapeStyle) {
	applyStyle(style.PrimitiveStyle)
	segments := style.Segments
	if segments == 0 {
		segments = 64
	}
	Circle(x, y, z, radius, style.Mode, segments)
}

// DrawPolygon zeichnet ein beliebiges 3D-Polygon als flaches Array
// [x0,y0,z0, x1,y1,z1, …] (Mode: 0=stroke | 1=fill | 2=beide).
func DrawPolygon(pts []float64, style ShapeStyle) {
	applyStyle(style.PrimitiveStyle)
	Polygon(pts, style.Mode)
}
