package main

import (
	"log"
	"runtime"
	"time"

	"lib3dgo/lib3d"
)

func main() {
	runtime.LockOSThread()

	if err := lib3d.Init(1600, 1000); err != nil {
		log.Fatal(err)
	}
	camPos := lib3d.Vec3{X: 50, Y: 20, Z: 200}
	target := lib3d.Vec3{X: 0, Y: 0, Z: 0}
	up := lib3d.Vec3{X: 0, Y: 1, Z: 0}
	lib3d.SetFog(100.0, 400.0, 0.25, 0.25, 0.25, 1.0)

	gridMesh := lib3d.CreateGridSolid(600, 24)
	grid := lib3d.NewBody(gridMesh, 0, 0, 0, &lib3d.BodyConfig{Color: "#777774", LineWidth: 1.0})

	vehicMesh := lib3d.CreatePyramidSolid(2, 6)
	vehics := make([]*Vehicle, 0, 10)
	for range 50 {
		vehic := NewVehicle(lib3d.NewBody(vehicMesh, 0, 20, 100, &lib3d.BodyConfigDefault))
		vehic.Body.Vel = lib3d.Vec3{X: lib3d.RandomFloat(-2, 2), Y: lib3d.RandomFloat(-2, 2), Z: lib3d.RandomFloat(-2, 2)}
		vehics = append(vehics, &vehic)
	}

	// Sieben schwebende Boxen als Hindernisse für die Boids.
	boxMesh := lib3d.CreateBoxSolid(40, 40, 40)
	obstacles := []*FloatingBox{
		NewFloatingBox(boxMesh, lib3d.Vec3{X: -90, Y: 40, Z: -50}, "#e06c5a", 0.0, 0.35),
		NewFloatingBox(boxMesh, lib3d.Vec3{X: 60, Y: 70, Z: 20}, "#5aa9e0", 1.7, -0.25),
		NewFloatingBox(boxMesh, lib3d.Vec3{X: 10, Y: 60, Z: -90}, "#8ad06a", 3.1, 0.2),
		NewFloatingBox(boxMesh, lib3d.Vec3{X: 80, Y: 0, Z: -40}, "#ff6b6b", 2.5, 0.15),
		NewFloatingBox(boxMesh, lib3d.Vec3{X: -40, Y: 100, Z: 40}, "#4ecdc4", 4.2, -0.3),
		NewFloatingBox(boxMesh, lib3d.Vec3{X: -110, Y: 80, Z: 80}, "#f4a259", 5.5, 0.28),
		NewFloatingBox(boxMesh, lib3d.Vec3{X: 120, Y: 50, Z: -80}, "#b07ce8", 0.9, -0.22),
	}
	const obstacleMargin = 20.0

	start := time.Now()

	/*---------------------------------
	Render-Schleife (in der Library)
	---------------------------------*/
	draw := func() {
		lib3d.Background(40, 40, 40)

		view := lib3d.LookAtMatrix(camPos, target, up)
		proj := lib3d.PerspectiveMatrix(1.2, float64(lib3d.GetWidth())/float64(lib3d.GetHeight()), 0.1, 1000.0)
		lib3d.SetProjection(proj)
		lib3d.SetView(view)

		sunDir := lib3d.Vec3{X: 0.5, Y: 1.0, Z: 0.3}
		camLight := sunDir.TransformDir(view)
		lib3d.SetLightDirection(camLight.X, camLight.Y, camLight.Z)

		lib3d.DrawBody(grid)

		// Hindernisse animieren und zeichnen.
		now := time.Since(start).Seconds()
		for _, o := range obstacles {
			o.Update(now)
			lib3d.DrawBody(o.Body)
		}

		for _, v := range vehics {
			if lib3d.IsMouseDown() {
				v.Seek(lib3d.Vec3{X: 0, Y: 0, Z: 0})
			}
			v.Allign(vehics)
			v.Separate(vehics)
			v.Cohesion(vehics)
			v.ApplyBoundary()
			v.AvoidObstacles(obstacles, obstacleMargin)
			v.AlignToVelocity()
			v.Update()
			lib3d.DrawBody(v.Body)
		}
	}

	lib3d.StartAnimation(draw)
}
