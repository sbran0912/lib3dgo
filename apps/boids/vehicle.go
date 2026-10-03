package main

import (
	"math"

	"lib3dgo/lib3d"
)

// Vehicle ist ein autonomes Fahrzeug der Simulation.
type Vehicle struct {
	Body    *lib3d.Body
	Accel   lib3d.Vec3
	Heading lib3d.Vec3
	Health  float64
	DNA     [4]float64
}

// NewVehicle erstellt ein neues Vehicle mit zufälliger DNA.
func NewVehicle(body *lib3d.Body) Vehicle {
	return Vehicle{
		Body:    body,
		Accel:   lib3d.Vec3{},
		Heading: lib3d.Vec3{},
	}
}

// AlignToVelocity richtet den Body an der Geschwindigkeit aus.
func (v *Vehicle) AlignToVelocity() {
	vel := v.Body.Vel

	mag := math.Sqrt(vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z)
	if mag < 0.0001 {
		return
	}

	magXZ := math.Sqrt(vel.X*vel.X + vel.Z*vel.Z)

	// vel.Y/mag muss in [-1,1] liegen (Schutz vor NaN durch Float-Rundung)
	cosY := lib3d.Constrain(vel.Y/mag, -1, 1)
	v.Body.RotX = math.Acos(cosY)
	if magXZ < 0.0001 {
		v.Body.RotY = 0
	} else {
		v.Body.RotY = math.Atan2(vel.X/magXZ, vel.Z/magXZ)
	}
	v.Body.RotZ = 0

	// heading als normalisierte Richtung ableiten
	v.Heading = lib3d.Vec3{X: vel.X / mag, Y: vel.Y / mag, Z: vel.Z / mag}
}

// ApplyForce addiert eine Kraft zur Beschleunigung.
func (v *Vehicle) ApplyForce(force lib3d.Vec3) {
	v.Accel = v.Accel.Add(force)
}

// Update integriert die Bewegung und begrenzt die Geschwindigkeit.
func (v *Vehicle) Update() {
	v.Body.Vel = v.Body.Vel.Add(v.Accel)
	speed := v.Body.Vel.Length()
	v.Body.Vel = v.Body.Vel.Normalize().Scale(lib3d.Constrain(speed, 0.5, 2.0))

	v.Accel = lib3d.Vec3{}
	v.Body.Pos = v.Body.Pos.Add(v.Body.Vel)
}

// Seek steuert das Fahrzeug in Richtung eines Ziels.
func (v *Vehicle) Seek(target lib3d.Vec3) {
	desired := target.Sub(v.Body.Pos).Limit(3.0)
	steer := desired.Sub(v.Body.Vel).Limit(2.0)
	v.ApplyForce(steer.Scale(0.2))
}

// Allign gleicht die eigene Geschwindigkeit an die der Nachbarn an
// (Flocking-Regel: Alignment).
func (v *Vehicle) Allign(vehics []*Vehicle) {
	const minDistance = 50.0
	sumVel := lib3d.Vec3{}
	count := 0

	for _, other := range vehics {
		distance := v.Body.Pos.DistanceTo(other.Body.Pos)
		if distance > 0 && distance < minDistance {
			sumVel = sumVel.Add(other.Body.Vel)
			count++
		}
	}

	if count > 0 {
		sumVel = sumVel.Scale(1 / float64(count)) // Durchschnittsgeschwindigkeit
		sumVel = sumVel.Mag(0.1)
		steer := sumVel.Sub(v.Body.Vel).Scale(0.1)
		v.ApplyForce(steer)
	}
}

func (v *Vehicle) Separate(vehics []*Vehicle) {
	const minDistance = 40.0
	diffSum := lib3d.Vec3{}
	count := 0
	for _, other := range vehics {
		distance := v.Body.Pos.DistanceTo(other.Body.Pos)
		if distance > 0 && distance < minDistance {
			diff := v.Body.Pos.Sub(other.Body.Pos).Normalize()
			diffSum = diffSum.Add(diff)
			count++
		}
	}
	if count > 0 {
		diffSum = diffSum.Scale(1 / float64(count)) // Durchschnitt
		diffSum = diffSum.Mag(0.1)
		steer := diffSum.Sub(v.Body.Vel).Scale(0.1)
		v.ApplyForce(steer)
	}
}

// Cohesion steuert in Richtung des Schwerpunkts der Nachbarn
// (Flocking-Regel: Kohäsion).
func (v *Vehicle) Cohesion(vehics []*Vehicle) {
	const minDistance = 50.0
	sumPos := lib3d.Vec3{}
	count := 0

	for _, other := range vehics {
		distance := v.Body.Pos.DistanceTo(other.Body.Pos)
		if distance > 0 && distance < minDistance {
			sumPos = sumPos.Add(other.Body.Pos)
			count++
		}
	}

	if count > 0 {
		sumPos = sumPos.Scale(1 / float64(count)) // Schwerpunkt der Nachbarn
		desired := sumPos.Sub(v.Body.Pos)         // Richtung zur Gruppenmitte
		desired = desired.Mag(0.1)
		steer := desired.Sub(v.Body.Vel).Scale(0.1)
		v.ApplyForce(steer)
	}
}

// ApplyBoundary reflektiert die Geschwindigkeit an den Weltgrenzen.
func (v *Vehicle) ApplyBoundary() {
	// Definiere deine Weltgrenzen
	const (
		minX = -200.0
		maxX = 200.0
		minY = -130.0
		maxY = 130.0
		minZ = -130.0
		maxZ = 130.0
	)

	// X-Achse
	if v.Body.Pos.X < minX || v.Body.Pos.X > maxX {
		v.Body.Vel.X *= -1.0
	}

	// Y-Achse (Höhe)
	if v.Body.Pos.Y < minY || v.Body.Pos.Y > maxY {
		v.Body.Vel.Y *= -1.0
	}

	// Z-Achse (Tiefe)
	if v.Body.Pos.Z < minZ || v.Body.Pos.Z > maxZ {
		v.Body.Vel.Z *= -1.0
	}
}

// AvoidObstacles hält das Vehicle mit einem Sicherheitsabstand (margin)
// außerhalb der schwebenden Boxen. Die Kollisionserkennung nutzt die konvexen
// Flächen-Ebenen des Hindernisses (Body.GetFacePlanes) und einen
// Abstands-Test auf Basis der Ebenengleichung.
//
// Der vorzeichenbehaftete Abstand n·p + d liefert den Abstand zur Fläche;
// die Projektion wird über IsPointInConvexPolygon darauf geprüft, ob sie
// innerhalb der Fläche liegt. Damit lässt sich die echte Distanz zur nächsten
// Fläche bestimmen. Liegt der Punkt vor einer Kante/Ecke, trifft keine Fläche
// – dann dient der größte vorzeichenbehaftete Ebenenabstand als konservative
// Untergrenze des Abstands.
//
// Unterschreitet der Abstand margin, wird das Vehicle entlang der Außen-Normalen
// herausgeschoben und die Geschwindigkeitskomponente ins Hindernis entfernt
// (Gleitbewegung).
func (v *Vehicle) AvoidObstacles(obstacles []*FloatingBox, margin float64) {
	for _, o := range obstacles {
		// Schneller Vorab-Test (Bounding Sphere): spart das Erzeugen der
		// Ebenen für weit entfernte Boids.
		if v.Body.Pos.DistanceTo(o.Body.Pos) > o.Radius+margin {
			continue
		}

		planes := o.Body.GetFacePlanes()
		if len(planes) == 0 {
			continue
		}

		// Für alle Flächen den vorzeichenbehafteten Abstand berechnen.
		//
		// Außen: Die getroffene Fläche mit dem kleinsten positiven Abstand
		//        ist die nächste Oberfläche (nearest).
		// Innen: Alle Abstände sind negativ; die betragsmäßig kleinste
		//        (also der GRÖSSTE signed distance) zeigt auf die Fläche, die
		//        zum Herauschieben am nächsten liegt.
		// Kanten/Ecken außen: keine Fläche enthält die Projektion → der
		//        größte signed distance dient als konservative Untergrenze.
		nearest := math.Inf(1)
		signed := 0.0
		normal := lib3d.Vec3{}
		deepest := math.Inf(-1)
		deepestNormal := lib3d.Vec3{}

		for _, p := range planes {
			// Signed distance zur Ebene (EINMAL berechnet).
			dist := p.Normal.Dot(v.Body.Pos) + p.Distance
			// Projektion auf die Fläche und Test, ob sie innerhalb liegt.
			proj := v.Body.Pos.Sub(p.Normal.Scale(dist))
			within := lib3d.IsPointInConvexPolygon(proj, p.Boundary, p.Normal)

			// Größter signed distance (Innen-Erkennung / Kanten-Fallback).
			if dist > deepest {
				deepest = dist
				deepestNormal = p.Normal
			}

			// Kürzester Abstand zu einer Fläche, die das Fahrzeug enthält
			// (nur außerhalb sinnvoll).
			if within && dist > 0 && dist < nearest {
				nearest = dist
				signed = dist // Wiederverwendung!
				normal = p.Normal
			}
		}

		// Innen (deepest <= 0) oder außen an Kante/Ecke ohne Flächentreffer:
		// konservativen Wert samt bester Aus-Schub-Richtung verwenden.
		if deepest <= 0 || math.IsInf(nearest, 1) {
			signed = deepest
			normal = deepestNormal
		}

		// signed < 0 → innerhalb des Volumens, sonst Abstand zur Oberfläche.
		// In beiden Fällen stellt eine Verschiebung um (margin - signed)
		// entlang der Außen-Normalen den Sicherheitsabstand her.
		if signed >= margin {
			continue
		}

		// Herauschieben: verhindert, dass das Boid eindringt und verschwindet.
		v.Body.Pos = v.Body.Pos.Add(normal.Scale(margin - signed))

		// Geschwindigkeit UND Beschleunigung ins Hindernis hinein streichen,
		// damit die Steering-Kräfte das Boid nicht sofort wieder hineindrücken.
		if into := v.Body.Vel.Dot(normal); into < 0 {
			v.Body.Vel = v.Body.Vel.Sub(normal.Scale(into))
		}
		if into := v.Accel.Dot(normal); into < 0 {
			v.Accel = v.Accel.Sub(normal.Scale(into))
		}
		v.ApplyForce(normal.Scale(0.01))
	}
}
