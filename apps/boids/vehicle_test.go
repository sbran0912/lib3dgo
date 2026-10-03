package main

import (
	"math"
	"testing"

	"lib3dgo/lib3d"
)

// TestAvoidObstaclesKeepsBoidOutside simuliert Boids, die mit einer
// konstanten Kraft in eine Box (bzw. deren Kante/Ecke) gedrückt werden, und
// prüft, dass sie die Box nie durchdringen. Bei der rotierenden Box wird
// zusätzlich auf numerische Stabilität (keine NaN-Position) geprüft.
func TestAvoidObstaclesKeepsBoidOutside(t *testing.T) {
	boxMesh := lib3d.CreateBoxSolid(40, 40, 40)
	const margin = 20.0

	tests := []struct {
		name  string
		spin  float64
		start lib3d.Vec3
		vel   lib3d.Vec3
		force lib3d.Vec3
	}{
		{
			name:  "front_seite",
			start: lib3d.Vec3{Z: 60},
			vel:   lib3d.Vec3{Z: -2},
			force: lib3d.Vec3{Z: -0.5},
		},
		{
			name:  "kante",
			start: lib3d.Vec3{X: 60, Y: 60},
			vel:   lib3d.Vec3{X: -2, Y: -2},
			force: lib3d.Vec3{X: -0.3, Y: -0.3},
		},
		{
			name:  "ecke",
			start: lib3d.Vec3{X: 60, Y: 60, Z: 60},
			vel:   lib3d.Vec3{X: -2, Y: -2, Z: -2},
			force: lib3d.Vec3{X: -0.2, Y: -0.2, Z: -0.2},
		},
		{
			name:  "rotierend",
			spin:  0.35,
			start: lib3d.Vec3{X: 50, Y: 30, Z: 70},
			vel:   lib3d.Vec3{X: -2, Y: -1, Z: -2},
			force: lib3d.Vec3{X: -0.3, Y: -0.1, Z: -0.3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obstacles := []*FloatingBox{
				NewFloatingBox(boxMesh, lib3d.Vec3{}, "#ffffff", 0, tc.spin),
			}

			body := lib3d.NewBody(lib3d.CreatePyramidSolid(2, 6), tc.start.X, tc.start.Y, tc.start.Z, &lib3d.BodyConfigDefault)
			body.Vel = tc.vel
			v := NewVehicle(body)

			for step := 0; step < 2000; step++ {
				obstacles[0].Update(float64(step) * 0.016)
				v.ApplyForce(tc.force)
				v.AvoidObstacles(obstacles, margin)
				v.Update()

				p := v.Body.Pos
				if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsNaN(p.Z) {
					t.Fatalf("Schritt %d: NaN-Position", step)
				}
				// Nur für die nicht-rotierende Box ist "innen" exakt prüfbar.
				if tc.spin == 0 {
					inside := math.Abs(p.X) < 20-0.5 && math.Abs(p.Y) < 20-0.5 && math.Abs(p.Z) < 20-0.5
					if inside {
						t.Fatalf("Schritt %d: Boid im Inneren der Box bei %+v", step, p)
					}
				}
			}
		})
	}
}
