package main

import (
	"log"
	"math"
	"runtime"

	"lib3dgo/lib3d"
)

// ====================================================================
// VEHICLE – Physik-fähiges Fahrzeug mit Steering Behaviors
// ====================================================================
type Vehicle struct {
	Body    *lib3d.Body
	Vel     lib3d.Vec3
	Accel   lib3d.Vec3
	Heading lib3d.Vec3
	Health  float64
	DNA     [4]float64
}

func newVehicle(body *lib3d.Body) *Vehicle {
	return &Vehicle{
		Body:    body,
		Vel:     lib3d.Vec3{},
		Accel:   lib3d.Vec3{},
		Heading: lib3d.Vec3{},
		Health:  1,
		DNA: [4]float64{
			lib3d.RandomFloat(-1.0, 1.0),  // DNA[0]: Kraft Richtung Gift
			lib3d.RandomFloat(-1.0, 1.0),  // DNA[1]: Kraft Richtung gutes Futter
			lib3d.RandomFloat(20.0, 60.0), // DNA[2]: Radius für Gift
			lib3d.RandomFloat(20.0, 60.0), // DNA[3]: Radius für gutes Futter
		},
	}
}

// AlignToVelocity passt die Richtung des Bodys an die Geschwindigkeit an.
func (v *Vehicle) AlignToVelocity() {
	vel := v.Vel
	mag := math.Sqrt(vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z)
	if mag < 0.0001 {
		return
	}

	magXZ := math.Sqrt(vel.X*vel.X + vel.Z*vel.Z)

	// vel.Y/mag muss in [-1,1] liegen (Schutz vor NaN durch Float-Rundung).
	cosY := lib3d.Constrain(vel.Y/mag, -1, 1)
	v.Body.RotX = math.Acos(cosY)
	if magXZ < 0.0001 {
		v.Body.RotY = 0
	} else {
		v.Body.RotY = math.Atan2(vel.X/magXZ, vel.Z/magXZ)
	}
	v.Body.RotZ = 0

	// Heading als normalisierte Richtung ableiten.
	v.Heading = lib3d.Vec3{X: vel.X / mag, Y: vel.Y / mag, Z: vel.Z / mag}
}

// ApplyForce wendet eine Kraft an (akkumuliert in Accel).
func (v *Vehicle) ApplyForce(force lib3d.Vec3) {
	v.Accel = v.Accel.Add(force)
}

// Update ist das Physik-Update: Geschwindigkeit aus Beschleunigung,
// Position aus Geschwindigkeit.
func (v *Vehicle) Update() {
	v.Vel = v.Vel.Add(v.Accel)
	speed := v.Vel.Length()
	// Geschwindigkeit auf [0.5, 2.0] begrenzen.
	v.Vel = v.Vel.Normalize().Scale(lib3d.Constrain(speed, 0.5, 2.0))
	v.Accel = lib3d.Vec3{}
	v.Body.Pos = v.Body.Pos.Add(v.Vel)
}

// Seek ist das Seek-Verhalten. Die gewünschte Richtung wird je nach Futter-Art
// mit dem passenden DNA-Wert skaliert (DNA[0] für Gift, DNA[1] für gutes Futter).
func (v *Vehicle) Seek(target lib3d.Vec3, isBadFood bool) {
	desired := target.Sub(v.Body.Pos).Limit(3)
	if isBadFood {
		desired = desired.Scale(v.DNA[0])
	} else {
		desired = desired.Scale(v.DNA[1])
	}
	steer := desired.Sub(v.Vel).Limit(2)
	v.ApplyForce(steer.Scale(0.2))
}

// ====================================================================
// HILFSFUNKTIONEN
// ====================================================================

// createFood erzeugt count Futter-Körper an Zufallspositionen (gemeinsames Mesh).
func createFood(count int, color string, mesh *lib3d.Solid) []*lib3d.Body {
	food := make([]*lib3d.Body, 0, count)
	for i := 0; i < count; i++ {
		food = append(food, lib3d.NewBody(mesh,
			float64(lib3d.Random(-100, 100)),
			float64(lib3d.Random(-100, 100)),
			float64(lib3d.Random(-100, 100)),
			&lib3d.BodyConfig{Color: color, LineWidth: 1}))
	}
	return food
}

// respawnFood füllt das Futter-Array auf: sind weniger als min vorhanden,
// kommen count neue dazu.
func respawnFood(food []*lib3d.Body, mesh *lib3d.Solid, min, count int, color string) []*lib3d.Body {
	if len(food) < min {
		for i := 0; i < count; i++ {
			food = append(food, lib3d.NewBody(mesh,
				float64(lib3d.Random(-100, 100)),
				float64(lib3d.Random(-100, 100)),
				float64(lib3d.Random(-100, 100)),
				&lib3d.BodyConfig{Color: color, LineWidth: 1}))
		}
	}
	return food
}

// vehicleEatFood sucht das nächste Futter im DNA-Radius und frisst es, wenn es
// nah genug ist. Gutes Futter → Health +0.1, Gift → Health -0.1. Ansonsten
// wird es angesteuert (Seek). Rückgabe: ggf. gekürztes food-Array.
func vehicleEatFood(vehic *Vehicle, food []*lib3d.Body, isBadFood bool) []*lib3d.Body {
	filter := vehic.DNA[3]
	if isBadFood {
		filter = vehic.DNA[2]
	}
	mindist := math.Inf(1)
	idx := -1

	for i := 0; i < len(food); i++ {
		distance := vehic.Body.Pos.DistanceTo(food[i].Pos)
		if distance < filter && distance < mindist {
			mindist = distance
			idx = i
		}
	}

	if idx > -1 {
		if vehic.Body.Pos.DistanceTo(food[idx].Pos) < 3 {
			// Food aufessen: aus dem Array entfernen genügt – es gibt keinen
			// GPU-Buffer/Refcount mehr freizugeben (nur CPU-Geometrie).
			food = append(food[:idx], food[idx+1:]...)
			if isBadFood {
				vehic.Health += -0.1
			} else {
				vehic.Health += 0.1
			}
		} else {
			vehic.Seek(food[idx].Pos, isBadFood)
		}
	}
	return food
}

// vehicBoundary: Geschwindigkeit an den Rändern umkehren (Bounce).
func vehicBoundary(vehic *Vehicle) {
	const (
		minX, maxX = -130.0, 130.0
		minY, maxY = -130.0, 130.0
		minZ, maxZ = -130.0, 130.0
	)
	if vehic.Body.Pos.X < minX || vehic.Body.Pos.X > maxX {
		vehic.Vel.X *= -1
	}
	if vehic.Body.Pos.Y < minY || vehic.Body.Pos.Y > maxY {
		vehic.Vel.Y *= -1
	}
	if vehic.Body.Pos.Z < minZ || vehic.Body.Pos.Z > maxZ {
		vehic.Vel.Z *= -1
	}
}

// vehicIsDead gibt true zurück, wenn das Vehicle tot ist (Health < 0).
func vehicIsDead(vehic *Vehicle) bool {
	return vehic.Health < 0
}

func main() {
	const (
		screenW = 1400
		screenH = 800
	)

	const (
		fovY  = 1.2
		zNear = 0.1
		zFar  = 1000
	)

	var (
		camPos    = lib3d.Vec3{X: 50, Y: 20, Z: 200}
		camTarget = lib3d.Vec3{X: 0, Y: 0, Z: 0}
		camUp     = lib3d.Vec3{X: 0, Y: 1, Z: 0}
	)

	// Weltfeste Lichtrichtung („Sonne“).
	sunDir := lib3d.Vec3{X: 0.5, Y: 1.0, Z: 0.3}

	// Bodengitter (CreateGridSolid(600, 24))
	grid := lib3d.CreateGrid(600, 24, 0, 0, 0, &lib3d.BodyConfig{Color: "#777774", LineWidth: 1})

	// Futter-Mesh: Kugel mit Radius 3 (CreateSphereSolid(3, 8, 8))
	foodMesh := lib3d.CreateSphereSolid(3, 8, 8)

	// 30× Gift (rot) und 30× gutes Futter (grün)
	poison := createFood(30, "#FF0000", foodMesh)
	food := createFood(30, "#44ff44", foodMesh)

	// Vehicle-Mesh: Pyramide (CreatePyramidSolid(2, 6))
	vehicleMesh := lib3d.CreatePyramidSolid(2, 6)

	// 10 Fahrzeuge mit zufälliger Startgeschwindigkeit
	var vehicles []*Vehicle
	for i := 0; i < 10; i++ {
		vehic := newVehicle(lib3d.NewBody(vehicleMesh, 0, 20, 100, nil))
		vehic.Vel = lib3d.Vec3{
			X: lib3d.RandomFloat(-2, 2),
			Y: lib3d.RandomFloat(-2, 2),
			Z: lib3d.RandomFloat(-2, 2),
		}
		vehicles = append(vehicles, vehic)
	}

	/* Loop */
	draw := func() {
		lib3d.Background(40, 40, 40)

		view := lib3d.LookAtMatrix(camPos, camTarget, camUp)
		proj := lib3d.PerspectiveMatrix(fovY, float64(lib3d.GetWidth())/float64(lib3d.GetHeight()), zNear, zFar)
		lib3d.SetProjection(proj)
		lib3d.SetView(view)

		// Weltfeste „Sonne“: Die Richtung ist im Weltraum fix und wird pro Frame
		// in den Kameraraum gedreht.
		camLight := sunDir.TransformDir(view)
		lib3d.SetLightDirection(camLight.X, camLight.Y, camLight.Z)

		// Bodengitter
		lib3d.DrawBody(grid)

		// Futter nachwachsen lassen (min 20, +30 pro Respawn)
		food = respawnFood(food, foodMesh, 20, 30, "#44ff44")
		poison = respawnFood(poison, foodMesh, 20, 30, "#FF0000")

		// Altern: mit 1.5% Wahrscheinlichkeit pro Frame altern alle Fahrzeuge
		getOlder := lib3d.RandomFloat(0, 1) < 0.015

		// Fahrzeuge simulieren (rückwärts, damit Entfernen beim Iterieren ok ist)
		for i := len(vehicles) - 1; i >= 0; i-- {
			v := vehicles[i]

			vehicBoundary(v)
			food = vehicleEatFood(v, food, false)    // gutes Futter
			poison = vehicleEatFood(v, poison, true) // Gift
			v.AlignToVelocity()
			v.Update()

			// Farbe nach Gesundheit: rot wenn schwach, sonst weiß
			if v.Health < 0.5 {
				v.Body.Color = "#FF0000"
			} else {
				v.Body.Color = "#ffffff"
			}

			lib3d.DrawBody(v.Body)

			if getOlder {
				v.Health -= 0.05
			}

			if vehicIsDead(v) {
				// Kein GPU-Aufräumen nötig – der Body hält nur CPU-Geometrie.
				vehicles = append(vehicles[:i], vehicles[i+1:]...)
			}
		}

		// Futter zeichnen
		for _, p := range poison {
			lib3d.DrawBody(p)
		}
		for _, f := range food {
			lib3d.DrawBody(f)
		}

		// Debug-Overlays (Batched Drawing): Heading-Pfeile + DNA-Radien.
		for _, v := range vehicles {
			hp := v.Body.Pos

			// Heading-Pfeil: Linie vom Fahrzeug in Fahrtrichtung.
			end := hp.Add(v.Heading.Scale(8))
			arrowColor := interface{}("#ffffff")
			if v.Health < 0.5 {
				arrowColor = "#ff4444"
			}
			lib3d.DrawLine(hp.X, hp.Y, hp.Z, end.X, end.Y, end.Z,
				lib3d.PrimitiveStyle{Stroke: arrowColor})

			// DNA-Radien als Kreise in der XZ-Ebene. DrawCircle(x,y,z) zeichnet in
			// der XY-Ebene – durch den Tausch (x, z, y) liegt der Kreis flach.
			lib3d.DrawCircle(hp.X, hp.Z, hp.Y, v.DNA[3], lib3d.ShapeStyle{
				PrimitiveStyle: lib3d.PrimitiveStyle{Stroke: []int{51, 255, 51, 64}},
				Segments:       48,
			}) // guter Food-Radius
			lib3d.DrawCircle(hp.X, hp.Z, hp.Y, v.DNA[2], lib3d.ShapeStyle{
				PrimitiveStyle: lib3d.PrimitiveStyle{Stroke: []int{255, 51, 51, 64}},
				Segments:       48,
			}) // Gift-Radius
		}
	}

	/* Start */
	runtime.LockOSThread()
	if err := lib3d.Init(screenW, screenH); err != nil {
		log.Fatal(err)
	}
	lib3d.SetFog(100, 400, 0.25, 0.25, 0.25, 1) // wie main vehic.ts: SetFog(100, 400, …)
	lib3d.StartAnimation(draw)
}
