package lib3d

import (
	"math"
	"testing"
)

func nearly(a, b, eps float64) bool { return math.Abs(a-b) < eps }

func TestVec3CrossAndDot(t *testing.T) {
	x := Vec3{1, 0, 0}
	y := Vec3{0, 1, 0}
	if got := x.Cross(y); !got.Equals(Vec3{0, 0, 1}, 1e-12) {
		t.Fatalf("cross = %+v, want (0,0,1)", got)
	}
	if got := x.Dot(y); got != 0 {
		t.Fatalf("dot = %v, want 0", got)
	}
}

func TestVec3Normalize(t *testing.T) {
	got := (Vec3{3, 0, 4}).Normalize()
	if !nearly(got.Length(), 1, 1e-12) || !got.Equals(Vec3{0.6, 0, 0.8}, 1e-12) {
		t.Fatalf("normalize = %+v", got)
	}
	if zero := (Vec3{}).Normalize(); zero != (Vec3{}) {
		t.Fatalf("zero vector normalize = %+v, want zero", zero)
	}
}

func TestPlaneIntersectGround(t *testing.T) {
	ground := NewPlane(Vec3{0, 1, 0}, 0, nil)
	hit, ok := ground.IntersectLine(Vec3{0, 5, 0}, Vec3{0, -5, 0})
	if !ok || !hit.Equals(Vec3{0, 0, 0}, 1e-12) {
		t.Fatalf("hit = %+v ok=%v, want (0,0,0)", hit, ok)
	}
	// Parallel → kein Treffer.
	if _, ok := ground.IntersectLine(Vec3{0, 1, 0}, Vec3{1, 1, 0}); ok {
		t.Fatal("parallele Strecke sollte die Ebene nicht schneiden")
	}
}

func TestPlaneBoundaryRejectsOutside(t *testing.T) {
	// Quadrat in der XZ-Ebene von (-1..1), CCW bzgl. der Normale +Y.
	boundary := []Vec3{
		{-1, 0, -1}, {-1, 0, 1}, {1, 0, 1}, {1, 0, -1},
	}
	plane := NewPlane(Vec3{0, 1, 0}, 0, boundary)
	if _, ok := plane.IntersectLine(Vec3{0, 5, 0}, Vec3{0, -5, 0}); !ok {
		t.Fatal("Treffer innerhalb des Polygons erwartet")
	}
	if _, ok := plane.IntersectLine(Vec3{5, 5, 0}, Vec3{5, -5, 0}); ok {
		t.Fatal("Treffer außerhalb des Polygons sollte abgelehnt werden")
	}
}

func TestCreatePlaneFromFace(t *testing.T) {
	// Vorderseite einer Box bei z=-hd: Normale zeigt in -Z.
	box := CreateBoxSolid(2, 2, 2)
	face := []Vec3{box.Vertices[0], box.Vertices[1], box.Vertices[2]}
	p := CreatePlaneFromFace(face)
	if !(p.Normal.Equals(Vec3{0, 0, -1}, 1e-12) || p.Normal.Equals(Vec3{0, 0, 1}, 1e-12)) {
		t.Fatalf("Normale = %+v, erwartet ±(0,0,1)", p.Normal)
	}
	// Der Ebenenpunkt (erster Vertex) muss die Ebenengleichung erfüllen.
	if d := p.Normal.Dot(face[0]) + p.Distance; !nearly(d, 0, 1e-12) {
		t.Fatalf("Ebenengleichung verletzt: %v", d)
	}
}

func TestLookAtCameraMapsToOrigin(t *testing.T) {
	cam := Vec3{0, 0, -100}
	view := LookAtMatrix(cam, Vec3{0, 0, 0}, Vec3{0, 1, 0})
	got := cam.Transform(view)
	if !got.Equals(Vec3{0, 0, 0}, 1e-9) {
		t.Fatalf("Kamera → %+v, erwartet Ursprung", got)
	}
	// Ein Punkt 10 Einheiten vor der Kamera liegt bei z=10 (Kamera blickt +Z).
	ahead := Vec3{0, 0, -90}.Transform(view)
	if !nearly(ahead.Z, 10, 1e-9) {
		t.Fatalf("Punkt vor Kamera z = %v, erwartet 10", ahead.Z)
	}
}

func TestPerspectiveClipspace(t *testing.T) {
	proj := PerspectiveMatrix(math.Pi/4, 1, 0.1, 1000)
	// Ein Punkt auf der near-Ebene (z = near) muss z_clip = -w ergeben.
	near := 0.1
	v := Vec3{0, 0, near}.Transform(proj)
	if !nearly(v.Z/near, -1, 1e-9) {
		t.Fatalf("near-Ebene z_clip/w = %v, erwartet -1", v.Z/near)
	}
}

func TestSolidTopology(t *testing.T) {
	box := CreateBoxSolid(100, 80, 60)
	if len(box.Vertices) != 8 || len(box.Edges) != 12 || len(box.Faces) != 6 {
		t.Fatalf("Box: v=%d e=%d f=%d", len(box.Vertices), len(box.Edges), len(box.Faces))
	}
	// Pro Quad-Face 2 Dreiecke à 9 Floats = 18.
	if len(box.FlatFaces) != 6*18 {
		t.Fatalf("Box FlatFaces = %d, erwartet %d", len(box.FlatFaces), 6*18)
	}
	if len(box.FlatEdges) != 12*6 {
		t.Fatalf("Box FlatEdges = %d, erwartet %d", len(box.FlatEdges), 12*6)
	}

	sphere := CreateSphereSolid(3, 8, 8)
	if len(sphere.Vertices) != 81 || len(sphere.Faces) != 64 {
		t.Fatalf("Sphere: v=%d f=%d", len(sphere.Vertices), len(sphere.Faces))
	}

	grid := CreateGridSolid(600, 24)
	if len(grid.Vertices) != 25*25 || grid.Faces != nil {
		t.Fatalf("Grid: v=%d faces=%v", len(grid.Vertices), grid.Faces)
	}
}

func TestBodyModelMatrixAndCollision(t *testing.T) {
	b := CreateBox(10, 10, 10, 5, 0, 0, nil)
	planes := b.GetFacePlanes()
	if len(planes) != 6 {
		t.Fatalf("Box-Face-Planes = %d, erwartet 6", len(planes))
	}
	// Die verschobene Box hat eine Fläche bei x = 0 und eine bei x = 10.
	foundLeft, foundRight := false, false
	for _, p := range planes {
		if p.Normal.Equals(Vec3{-1, 0, 0}, 1e-12) && nearly(p.Distance, 0, 1e-12) {
			foundLeft = true
		}
		if p.Normal.Equals(Vec3{1, 0, 0}, 1e-12) && nearly(p.Distance, -10, 1e-12) {
			foundRight = true
		}
	}
	if !foundLeft || !foundRight {
		t.Fatalf("Box-Seitenflächen nicht korrekt (left=%v right=%v)", foundLeft, foundRight)
	}

	sphere := CreateSphere(3, 8, 8, 0, 0, 0, nil)
	if sphere.Faces != nil {
		t.Fatal("Kugel-Kollision sollte abgeschaltet sein")
	}
}

func TestParseColor(t *testing.T) {
	hex := ParseColor("#ff0000")
	if !nearly(hex.R, 1, 1e-12) || hex.G != 0 || hex.B != 0 || hex.A != 1 {
		t.Fatalf("hex = %+v", hex)
	}
	gray := ParseColor(128)
	if !nearly(gray.R, 128.0/255, 1e-12) || !nearly(gray.G, 128.0/255, 1e-12) {
		t.Fatalf("gray = %+v", gray)
	}
	rgba := ParseColor(51, 255, 51, 64)
	if !nearly(rgba.G, 1, 1e-12) || !nearly(rgba.A, 64.0/255, 1e-12) {
		t.Fatalf("rgba = %+v", rgba)
	}
}
