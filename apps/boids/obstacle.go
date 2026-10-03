package main

import (
	"math"

	"lib3dgo/lib3d"
)

// FloatingBox ist ein schwebendes Hindernis für die Boids: ein Body mit
// Grundposition, sanfter Auf-und-Ab-Bewegung und langsamer Rotation.
type FloatingBox struct {
	Body   *lib3d.Body
	Base   lib3d.Vec3
	Phase  float64
	Spin   float64
	Radius float64 // Bounding-Sphere-Radius für den schnellen Vorab-Test
}

// NewFloatingBox erstellt eine schwebende Hindernis-Box an Position pos.
func NewFloatingBox(mesh *lib3d.Solid, pos lib3d.Vec3, color string, phase, spin float64) *FloatingBox {
	radius := 0.0
	for _, vert := range mesh.Vertices {
		if l := vert.Length(); l > radius {
			radius = l
		}
	}

	body := lib3d.NewBody(mesh, pos.X, pos.Y, pos.Z, &lib3d.BodyConfig{Color: color, LineWidth: 1.5})
	return &FloatingBox{
		Body:   body,
		Base:   pos,
		Phase:  phase,
		Spin:   spin,
		Radius: radius,
	}
}

// Update animiert die Box (Schweben + Rotation). t ist die Zeit in Sekunden.
func (f *FloatingBox) Update(t float64) {
	f.Body.Pos = lib3d.Vec3{
		X: f.Base.X,
		Y: f.Base.Y + math.Sin(t+f.Phase)*6,
		Z: f.Base.Z,
	}
	f.Body.RotY = t * f.Spin
	f.Body.RotX = math.Sin(t*0.6+f.Phase) * 0.3
}
