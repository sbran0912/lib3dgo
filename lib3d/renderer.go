package lib3d

/*
-----------------------------------------------------------------
OpenGL-Implementierung – Echter 3D-Renderer (GPU-Transformation)
-----------------------------------------------------------------
Port von lib-wgl.ts (WebGL1) auf OpenGL 3.3 Core (go-gl).

Koordinatensystem: +Y zeigt nach oben, Kamera blickt in +Z.
Die 3D-Transformation (Model/View/Projection) läuft im Vertex-Shader.
Depth-Testing ist aktiviert.

Shader-Modi (SetEffect):
  EffectFlat      – einfache Volltonfarbe  (Standard)
  EffectGradient  – radialer Verlauf von Farbe A nach Farbe B
  EffectPulse     – pulsierende Helligkeit über die Zeit

Beleuchtung (Flat Shading):
  Alle gefüllten Flächen (fill) werden automatisch beleuchtet (uLighted=1),
  Kanten/Punkte/Linien (stroke) nicht (uLighted=0). Die Lichtrichtung setzt
  SetLightDirection() in Kamerakoordinaten – Default: Headlight aus der
  Kamera (uLightDir = 0,0,-1, da die Kamera in +Z blickt).
  Die Facetten-Normale wird pro Pixel aus den Kamera-Raum-Ableitungen
  (dFdx/dFdy) rekonstruiert → kein Normalen-Attribut nötig.

BATCHED DRAWING:
  Alle Immediate-Primitives eines Frames (Punkte, Linien, Kreise, aber auch
  die Solid-Meshes) werden nur GESAMMELT: Submit() hängt die Vertices an den
  Frame-Batch und speichert einen Zustands-Snapshot (DrawCmd). Aufeinander-
  folgende Befehle mit identischem Zustand werden zu einem DrawArrays-Aufruf
  verschmolzen (LINE_LOOP nie). Erst FlushBatch() (Frame-Ende bzw. vor
  Background()) lädt alles in EIN wiederverwendbares VBO und zeichnet mit
  pro Befehl gesetzten Uniforms.
-----------------------------------------------------------------
*/

import (
	"math"
	"runtime"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// =================================================================
// TYPEN & INTERNER ZUSTAND
// =================================================================

// DrawStyle: 0 = stroke | 1 = fill | 2 = beide.
type DrawStyle int

const (
	StyleStroke DrawStyle = 0
	StyleFill   DrawStyle = 1
	StyleBoth   DrawStyle = 2
)

// EffectMode ist der aktive Shader-Effekt.
type EffectMode int

const (
	EffectFlat EffectMode = iota
	EffectGradient
	EffectPulse
)

// DrawState ist der aktuelle Zeichenzustand (wird pro draw*-Aufruf
// überschrieben; kein Stack).
type DrawState struct {
	fill   ColorState
	stroke ColorState
	lineW  float64
	effect EffectMode
	grad2  ColorState
}

// DrawCmd ist ein gesammelter Zeichenbefehl für eine Immediate-Primitive.
// Alle Befehle eines Frames werden gepuffert und am Frame-Ende mit einem
// einzigen wiederverwendbaren VBO gezeichnet.
type DrawCmd struct {
	mode      uint32 // GL-Modus (gl.POINTS/LINES/TRIANGLES/LINE_LOOP)
	first     int    // Start-Index in batchVerts (Vertex-Nummer)
	count     int    // Anzahl Vertices
	col       ColorState
	effect    EffectMode
	grad2     ColorState
	pointSize float64
	lineW     float64
	center    [3]float64
	radius    float64
	view      [16]float32 // View-Snapshot (column-major)
	model     [16]float32 // Model-Snapshot (column-major)
	proj      [16]float32 // Projection-Snapshot
	lit       int32       // 1 = gefüllte Fläche → beleuchten
}

// Fenster / GL
var (
	window *glfw.Window
	prog   uint32
	vao    uint32
	vbo    uint32

	frameW, frameH int
)

// Shader-Locations
var (
	locPos        int32
	locView       int32
	locModel      int32
	locProjection int32
	locPointSize  int32
	locMode       int32
	locColor      int32
	locColor2     int32
	locTime       int32
	locCenter     int32
	locRadius     int32
	locFogNear    int32
	locFogFar     int32
	locFogColor   int32
	locLighted    int32
	locLightDir   int32
)

// Animation
var (
	looping   = true
	startTime float64
)

// Maus
var (
	mouseX      float64
	mouseY      float64
	mouseStatus = 0
)

// Aktueller Zeichenzustand.
var state = DrawState{
	fill:   ColorState{1, 1, 1, 1},
	stroke: ColorState{0, 0, 0, 1},
	lineW:  1,
	effect: EffectFlat,
	grad2:  ColorState{0, 0, 0, 1},
}

// Batching: gesammelte Immediate-Primitives eines Frames.
var (
	batchVerts []float32
	batchCmds  []DrawCmd
	batchCap   int // aktuell allozierte Byte-Größe des VBO
)

// Zuletzt gesetzte Uniform-Werte (für den Batch-Snapshot pro Befehl).
var (
	viewUniform  = identity16()
	modelUniform = identity16()
	projUniform  = identity16()
	pointSizeVal = 4.0
	gradCenter   = [3]float64{0, 0, 0}
	gradRadius   = 1.0
)

// =================================================================
// HILFSFUNKTIONEN (intern)
// =================================================================

// identity16 liefert die 4x4-Identitätsmatrix als flaches column-major Array.
func identity16() [16]float32 {
	return [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// flattenMatrix wandelt eine row-major Matrix4x4 in ein column-major
// Float32-Array für OpenGL (transpose=false).
func flattenMatrix(m Matrix4x4) [16]float32 {
	return [16]float32{
		float32(m[0][0]), float32(m[1][0]), float32(m[2][0]), float32(m[3][0]),
		float32(m[0][1]), float32(m[1][1]), float32(m[2][1]), float32(m[3][1]),
		float32(m[0][2]), float32(m[1][2]), float32(m[2][2]), float32(m[3][2]),
		float32(m[0][3]), float32(m[1][3]), float32(m[2][3]), float32(m[3][3]),
	}
}

func effectToMode(effect EffectMode) int32 {
	switch effect {
	case EffectGradient:
		return 1
	case EffectPulse:
		return 2
	default:
		return 0
	}
}

func arrEq(a, b [16]float32) bool {
	for i := 0; i < 16; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// canMerge prüft, ob zwei aufeinanderfolgende Befehle identischen Zustand
// haben und zu einem DrawArrays-Aufruf zusammengefasst werden können.
// LINE_LOOP wird nie zusammengefasst, weil ein Loop sich selbst schließt.
func canMerge(a, b DrawCmd) bool {
	return a.mode == b.mode &&
		a.mode != gl.LINE_LOOP &&
		colEq(a.col, b.col) &&
		a.effect == b.effect &&
		colEq(a.grad2, b.grad2) &&
		a.pointSize == b.pointSize &&
		a.lineW == b.lineW &&
		a.center[0] == b.center[0] &&
		a.center[1] == b.center[1] &&
		a.center[2] == b.center[2] &&
		a.radius == b.radius &&
		arrEq(a.view, b.view) &&
		arrEq(a.model, b.model) &&
		arrEq(a.proj, b.proj) &&
		a.lit == b.lit
}

// =================================================================
// BATCHING – SAMMELN & ZEICHNEN (Kern des Renderers)
// =================================================================

// Submit sammelt eine Immediate-Primitive im Frame-Batch. Es wird noch nichts
// an die GPU geschickt; das passiert erst beim FlushBatch() (Frame-Ende bzw.
// vor Background()).
func Submit(mode uint32, verts []float32, useStroke bool) {
	col := state.fill
	lit := int32(1) // gefüllte Flächen werden beleuchtet
	if useStroke {
		col = state.stroke
		lit = 0
	}

	cmd := DrawCmd{
		mode:      mode,
		first:     len(batchVerts) / 3,
		count:     len(verts) / 3,
		col:       col,
		effect:    state.effect,
		grad2:     state.grad2,
		pointSize: pointSizeVal,
		lineW:     state.lineW,
		center:    gradCenter,
		radius:    gradRadius,
		view:      viewUniform,
		model:     modelUniform,
		proj:      projUniform,
		lit:       lit,
	}

	batchVerts = append(batchVerts, verts...)

	// Aufeinanderfolgende Befehle mit identischem Zustand zu einem
	// DrawArrays-Aufruf zusammenfassen (weniger Draw-Calls).
	n := len(batchCmds)
	if n > 0 && canMerge(batchCmds[n-1], cmd) {
		batchCmds[n-1].count += cmd.count
		return
	}
	batchCmds = append(batchCmds, cmd)
}

// SubmitTriangles sammelt gefüllte Dreiecke (Fill-Farbe, beleuchtet).
func SubmitTriangles(verts []float32) {
	Submit(gl.TRIANGLES, verts, false)
}

// SubmitLines sammelt Linien (Stroke-Farbe, unbelichtet).
func SubmitLines(verts []float32) {
	Submit(gl.LINES, verts, true)
}

// FlushBatch zeichnet alle gesammelten Primitives des Frames mit einem
// einzigen wiederverwendbaren VBO. Der GPU-Speicher wird nur beim Wachsen
// neu allokiert.
func FlushBatch() {
	if len(batchCmds) == 0 {
		return
	}

	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)

	byteLen := len(batchVerts) * 4
	if byteLen > batchCap {
		gl.BufferData(gl.ARRAY_BUFFER, byteLen, unsafe.Pointer(&batchVerts[0]), gl.DYNAMIC_DRAW)
		batchCap = byteLen
	} else {
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, byteLen, unsafe.Pointer(&batchVerts[0]))
	}

	gl.EnableVertexAttribArray(uint32(locPos))
	gl.VertexAttribPointer(uint32(locPos), 3, gl.FLOAT, false, 0, nil)

	for _, c := range batchCmds {
		gl.UniformMatrix4fv(locView, 1, false, &c.view[0])
		gl.UniformMatrix4fv(locModel, 1, false, &c.model[0])
		gl.UniformMatrix4fv(locProjection, 1, false, &c.proj[0])
		gl.Uniform1i(locMode, effectToMode(c.effect))
		gl.Uniform1i(locLighted, c.lit)
		gl.Uniform4f(locColor, float32(c.col.R), float32(c.col.G), float32(c.col.B), float32(c.col.A))
		gl.Uniform4f(locColor2, float32(c.grad2.R), float32(c.grad2.G), float32(c.grad2.B), float32(c.grad2.A))
		gl.Uniform1f(locPointSize, float32(c.pointSize))
		gl.Uniform3f(locCenter, float32(c.center[0]), float32(c.center[1]), float32(c.center[2]))
		gl.Uniform1f(locRadius, float32(c.radius))
		gl.LineWidth(float32(c.lineW))
		gl.DrawArrays(c.mode, int32(c.first), int32(c.count))
	}

	batchVerts = batchVerts[:0]
	batchCmds = batchCmds[:0]
}

// =================================================================
// MAUS / FENSTER (intern)
// =================================================================

func onCursorPos(_ *glfw.Window, x, y float64) {
	mouseX = x - float64(frameW)/2
	mouseY = -(y - float64(frameH)/2)
}

func onMouseButton(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
	if button != glfw.MouseButtonLeft {
		return
	}
	switch action {
	case glfw.Press:
		mouseStatus = 1
	case glfw.Release:
		mouseStatus = 2
	}
}

func onFramebufferSize(_ *glfw.Window, width, height int) {
	frameW, frameH = width, height
	gl.Viewport(0, 0, int32(width), int32(height))
}

// =================================================================
// ÖFFENTLICHE API – Setup & Matrizen
// =================================================================

// GetWidth liefert die aktuelle Framebuffer-Breite.
func GetWidth() int { return frameW }

// GetHeight liefert die aktuelle Framebuffer-Höhe.
func GetHeight() int { return frameH }

// NoLoop stoppt die Animations-Schleife.
func NoLoop() { looping = false }

// IsMouseDown gibt true zurück, solange die linke Maustaste gedrückt ist.
func IsMouseDown() bool { return mouseStatus == 1 }

// IsMouseUp gibt einmalig true zurück, wenn die Maustaste losgelassen wurde.
func IsMouseUp() bool {
	if mouseStatus == 2 {
		mouseStatus = 0
		return true
	}
	return false
}

// SetProjection setzt die Projection-Matrix für das 3D-Rendering.
func SetProjection(m Matrix4x4) {
	projUniform = flattenMatrix(m)
	gl.UniformMatrix4fv(locProjection, 1, false, &projUniform[0])
}

// SetView setzt die View-Matrix (Kamera) für das 3D-Rendering.
// Typischerweise einmal pro Frame. Der Wert wird pro Submit() als Snapshot
// in den Batch übernommen.
func SetView(m Matrix4x4) {
	viewUniform = flattenMatrix(m)
	gl.UniformMatrix4fv(locView, 1, false, &viewUniform[0])
}

// SetModel setzt die Model-Matrix (Objekt-Transformation) für das
// 3D-Rendering. Default ist die Identität. Der Wert wird pro Submit() als
// Snapshot in den Batch übernommen.
func SetModel(m Matrix4x4) {
	modelUniform = flattenMatrix(m)
	gl.UniformMatrix4fv(locModel, 1, false, &modelUniform[0])
}

// SetGradientCenter setzt das Zentrum für den Gradient-Effekt
// (in Kamera-Koordinaten).
func SetGradientCenter(cx, cy, cz, radius float64) {
	gradCenter = [3]float64{cx, cy, cz}
	gradRadius = radius
	gl.Uniform3f(locCenter, float32(cx), float32(cy), float32(cz))
	gl.Uniform1f(locRadius, float32(radius))
}

// SetFog aktiviert Tiefen-Nebel. Alle Objekte werden ab near immer mehr von
// der Nebelfarbe überdeckt, bis sie bei far vollständig darin verschwinden.
func SetFog(near, far, r, g, b, a float64) {
	gl.Uniform1f(locFogNear, float32(near))
	gl.Uniform1f(locFogFar, float32(far))
	gl.Uniform4f(locFogColor, float32(r), float32(g), float32(b), float32(a))
}

// SetLightDirection setzt die Lichtrichtung für das Flat-Shading in
// Kamerakoordinaten. Der Vektor zeigt vom Fragment ZUR Lichtquelle.
// Default nach Init(): Headlight aus der Kamera (0, 0, -1).
func SetLightDirection(x, y, z float64) {
	gl.Uniform3f(locLightDir, float32(x), float32(y), float32(z))
}

// Init initialisiert Fenster, OpenGL-Kontext, Shader und Batch-Zustand.
func Init(w, h int) error {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		return err
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Samples, 4)

	win, err := glfw.CreateWindow(w, h, "3D Renderer", nil, nil)
	if err != nil {
		glfw.Terminate()
		return err
	}
	window = win
	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.Init(); err != nil {
		glfw.Terminate()
		return err
	}

	program, err := createProgram(vertSrc, fragSrc)
	if err != nil {
		glfw.Terminate()
		return err
	}
	prog = program
	gl.UseProgram(prog)

	locPos = gl.GetAttribLocation(prog, gl.Str("aPos\x00"))
	locView = gl.GetUniformLocation(prog, gl.Str("uView\x00"))
	locModel = gl.GetUniformLocation(prog, gl.Str("uModel\x00"))
	locProjection = gl.GetUniformLocation(prog, gl.Str("uProjection\x00"))
	locPointSize = gl.GetUniformLocation(prog, gl.Str("uPointSize\x00"))
	locMode = gl.GetUniformLocation(prog, gl.Str("uMode\x00"))
	locColor = gl.GetUniformLocation(prog, gl.Str("uColor\x00"))
	locColor2 = gl.GetUniformLocation(prog, gl.Str("uColor2\x00"))
	locTime = gl.GetUniformLocation(prog, gl.Str("uTime\x00"))
	locCenter = gl.GetUniformLocation(prog, gl.Str("uShapeCenter\x00"))
	locRadius = gl.GetUniformLocation(prog, gl.Str("uShapeRadius\x00"))
	locFogNear = gl.GetUniformLocation(prog, gl.Str("uFogNear\x00"))
	locFogFar = gl.GetUniformLocation(prog, gl.Str("uFogFar\x00"))
	locFogColor = gl.GetUniformLocation(prog, gl.Str("uFogColor\x00"))
	locLighted = gl.GetUniformLocation(prog, gl.Str("uLighted\x00"))
	locLightDir = gl.GetUniformLocation(prog, gl.Str("uLightDir\x00"))

	// VAO/VBO anlegen (OpenGL 3.3 Core benötigt ein VAO).
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)

	gl.Uniform1f(locPointSize, 4.0)
	gl.Uniform3f(locCenter, 0, 0, 0)
	gl.Uniform1f(locRadius, 1)
	gl.Uniform1i(locLighted, 0)
	gl.Uniform3f(locLightDir, 0, 0, -1) // Headlight: Kamera blickt +Z → Licht aus Kamera = -Z

	// Default-Matrizen (Identität)
	ident := identity16()
	gl.UniformMatrix4fv(locProjection, 1, false, &ident[0])
	gl.UniformMatrix4fv(locView, 1, false, &ident[0])
	gl.UniformMatrix4fv(locModel, 1, false, &ident[0])

	// Fog-Defaults (kein Nebel)
	gl.Uniform1f(locFogNear, 0)
	gl.Uniform1f(locFogFar, 1)
	gl.Uniform4f(locFogColor, 0, 0, 0, 0)

	// Depth-Testing + Alpha-Blending.
	// LEQUAL (statt LESS): koplanare Kanten bleiben über ihren eigenen
	// Flächen sichtbar (sonst Z-Fighting zwischen Face-Füllung und Kanten).
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	// gl_PointSize im Vertex-Shader muss im Core-Profil aktiviert werden.
	gl.Enable(gl.PROGRAM_POINT_SIZE)

	frameW, frameH = w, h
	gl.Viewport(0, 0, int32(w), int32(h))
	startTime = glfw.GetTime()

	// Batch-Zustand initialisieren.
	state = DrawState{
		fill:   ColorState{1, 1, 1, 1},
		stroke: ColorState{0, 0, 0, 1},
		lineW:  1,
		effect: EffectFlat,
		grad2:  ColorState{0, 0, 0, 1},
	}
	viewUniform = identity16()
	modelUniform = identity16()
	projUniform = identity16()
	pointSizeVal = 4.0
	gradCenter = [3]float64{0, 0, 0}
	gradRadius = 1
	batchCap = 0
	batchVerts = batchVerts[:0]
	batchCmds = batchCmds[:0]

	// Eingabe-Callbacks.
	window.SetCursorPosCallback(onCursorPos)
	window.SetMouseButtonCallback(onMouseButton)
	window.SetFramebufferSizeCallback(onFramebufferSize)

	return nil
}

// StartAnimation startet die Animations-Schleife. Sie kapselt BeginFrame
// (uTime setzen) und EndFrame (Batch zeichnen), sodass der Aufrufer nur noch
// das Zeichnen bereitstellen muss.
func StartAnimation(fnDraw func()) {
	looping = true
	for looping && !window.ShouldClose() {
		// BeginFrame
		t := glfw.GetTime() - startTime
		gl.Uniform1f(locTime, float32(t))

		fnDraw()

		// EndFrame: alle gesammelten Primitives des Frames zeichnen
		FlushBatch()

		window.SwapBuffers()
		glfw.PollEvents()
	}
	glfw.Terminate()
}

// =================================================================
// ÖFFENTLICHE API – Zeichenzustand
// =================================================================

// FillColor setzt die Füllfarbe.
// Formen: Hex-String | Grau (0-255) | r,g,b | r,g,b,a (0-255).
func FillColor(color ...interface{}) {
	state.fill = ParseColor(color...)
}

// StrokeColor setzt die Linienfarbe.
// Formen: Hex-String | Grau (0-255) | r,g,b | r,g,b,a (0-255).
func StrokeColor(color ...interface{}) {
	state.stroke = ParseColor(color...)
}

// StrokeWidth setzt die Linienstärke in Pixeln.
func StrokeWidth(w float64) {
	state.lineW = w
}

// SetEffect wählt den aktiven Shader-Effekt.
func SetEffect(effect EffectMode) {
	state.effect = effect
}

// SetGradient setzt die zweite Farbe für den Gradient-Effekt.
func SetGradient(color ...interface{}) {
	state.grad2 = ParseColor(color...)
}

// Background löscht den gesamten Canvas inkl. Tiefenpuffer. Zeichnet zuerst
// noch nicht geflushte Primitives.
func Background(color ...interface{}) {
	FlushBatch()
	c := ParseColor(color...)
	gl.ClearColor(float32(c.R), float32(c.G), float32(c.B), float32(c.A))
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

// =================================================================
// ÖFFENTLICHE API – 3D-Primitive
// =================================================================

// PointSize setzt die Punktgröße in Pixeln.
func PointSize(px float64) {
	pointSizeVal = px
	gl.Uniform1f(locPointSize, float32(px))
}

// Point zeichnet einen 3D-Punkt bei (x,y,z).
func Point(x, y, z float64) {
	Submit(gl.POINTS, []float32{float32(x), float32(y), float32(z)}, true)
}

// Line zeichnet eine 3D-Linie von (x1,y1,z1) nach (x2,y2,z2).
func Line(x1, y1, z1, x2, y2, z2 float64) {
	Submit(gl.LINES, []float32{
		float32(x1), float32(y1), float32(z1),
		float32(x2), float32(y2), float32(z2),
	}, true)
}

// Triangle zeichnet ein 3D-Dreieck (style: 0=stroke | 1=fill | 2=beide).
func Triangle(x1, y1, z1, x2, y2, z2, x3, y3, z3 float64, style DrawStyle) {
	pts := []float32{
		float32(x1), float32(y1), float32(z1),
		float32(x2), float32(y2), float32(z2),
		float32(x3), float32(y3), float32(z3),
	}

	if style == StyleFill || style == StyleBoth {
		Submit(gl.TRIANGLES, pts, false)
	}
	if style == StyleStroke || style == StyleBoth {
		Submit(gl.LINES, []float32{
			float32(x1), float32(y1), float32(z1), float32(x2), float32(y2), float32(z2),
			float32(x2), float32(y2), float32(z2), float32(x3), float32(y3), float32(z3),
			float32(x3), float32(y3), float32(z3), float32(x1), float32(y1), float32(z1),
		}, true)
	}
}

// Shape zeichnet ein 3D-Viereck mit 4 beliebigen Punkten.
func Shape(x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4 float64, style DrawStyle) {
	if style == StyleFill || style == StyleBoth {
		Submit(gl.TRIANGLES, []float32{
			float32(x1), float32(y1), float32(z1), float32(x2), float32(y2), float32(z2), float32(x3), float32(y3), float32(z3),
			float32(x1), float32(y1), float32(z1), float32(x3), float32(y3), float32(z3), float32(x4), float32(y4), float32(z4),
		}, false)
	}
	if style == StyleStroke || style == StyleBoth {
		Submit(gl.LINES, []float32{
			float32(x1), float32(y1), float32(z1), float32(x2), float32(y2), float32(z2),
			float32(x2), float32(y2), float32(z2), float32(x3), float32(y3), float32(z3),
			float32(x3), float32(y3), float32(z3), float32(x4), float32(y4), float32(z4),
			float32(x4), float32(y4), float32(z4), float32(x1), float32(y1), float32(z1),
		}, true)
	}
}

// Rect zeichnet ein achsenparalleles Rechteck in der XY-Ebene bei z.
// (x,y) = Mittelpunkt, w×h in der XY-Ebene.
func Rect(x, y, w, h float64, style DrawStyle, z float64) {
	hw, hh := w/2, h/2
	Shape(x-hw, y-hh, z, x+hw, y-hh, z, x+hw, y+hh, z, x-hw, y+hh, z, style)
}

// Circle zeichnet einen 3D-Kreis (Ring) in der XY-Ebene bei z.
// (x,y,z) = Mittelpunkt, segments = Anzahl Dreiecks-Segmente.
func Circle(x, y, z, radius float64, style DrawStyle, segments int) {
	tau := 2 * math.Pi

	if style == StyleFill || style == StyleBoth {
		fillVerts := make([]float32, 0, segments*9)
		for i := 0; i < segments; i++ {
			a0 := (float64(i) / float64(segments)) * tau
			a1 := (float64(i+1) / float64(segments)) * tau
			fillVerts = append(fillVerts,
				float32(x), float32(y), float32(z),
				float32(x+math.Cos(a0)*radius), float32(y+math.Sin(a0)*radius), float32(z),
				float32(x+math.Cos(a1)*radius), float32(y+math.Sin(a1)*radius), float32(z),
			)
		}
		Submit(gl.TRIANGLES, fillVerts, false)
	}

	if style == StyleStroke || style == StyleBoth {
		strokeVerts := make([]float32, 0, segments*3)
		for i := 0; i < segments; i++ {
			a := (float64(i) / float64(segments)) * tau
			strokeVerts = append(strokeVerts,
				float32(x+math.Cos(a)*radius),
				float32(y+math.Sin(a)*radius),
				float32(z),
			)
		}
		Submit(gl.LINE_LOOP, strokeVerts, true)
	}
}

// Polygon zeichnet ein beliebiges 3D-Polygon als flaches Array
// [x0,y0,z0, x1,y1,z1, …] (style: 0=stroke | 1=fill | 2=beide).
// Hinweis: fill ist korrekt nur für konvexe Polygone (Fan-Triangulation).
func Polygon(pts []float64, style DrawStyle) {
	if len(pts) < 4 {
		return
	}
	n := len(pts) / 3

	if style == StyleFill || style == StyleBoth {
		fillVerts := make([]float32, 0)
		for i := 1; i < n-1; i++ {
			fillVerts = append(fillVerts,
				float32(pts[0]), float32(pts[1]), float32(pts[2]),
				float32(pts[i*3]), float32(pts[i*3+1]), float32(pts[i*3+2]),
				float32(pts[i*3+3]), float32(pts[i*3+4]), float32(pts[i*3+5]),
			)
		}
		Submit(gl.TRIANGLES, fillVerts, false)
	}
	if style == StyleStroke || style == StyleBoth {
		strokeVerts := make([]float32, len(pts))
		for i, v := range pts {
			strokeVerts[i] = float32(v)
		}
		Submit(gl.LINE_LOOP, strokeVerts, true)
	}
}
