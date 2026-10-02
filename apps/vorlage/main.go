package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"lib3dgo/lib3d"
)

func main() {
	const (
		screenW = 1200
		screenH = 800
	)

	const (
		fovY  = 1.2
		zNear = 0.1
		zFar  = 1000
	)

	var (
		camPos    = lib3d.Vec3{X: 50, Y: 100, Z: 200}
		camTarget = lib3d.Vec3{X: 0, Y: 0, Z: 0}
		camUp     = lib3d.Vec3{X: 0, Y: 1, Z: 0}
	)

	sunDir := lib3d.Vec3{X: 0.5, Y: 1.0, Z: 0.3}
	grid := lib3d.CreateGrid(600, 24, 0, 0, 0, &lib3d.BodyConfig{Color: "#777774", LineWidth: 1})
	ball := lib3d.CreateSphere(20, 16, 12, 0, 16, 0, &lib3d.BodyConfig{Color: "#d3ff44", LineWidth: 1})

	var (
		fpsFrames   int
		fpsLastTime = time.Now()
	)

	updateFps := func() {
		fpsFrames++
		now := time.Now()
		elapsed := now.Sub(fpsLastTime)
		if elapsed >= 500*time.Millisecond {
			fmt.Printf("FPS: %d\n", fpsFrames*1000/int(elapsed.Milliseconds()))
			fpsFrames = 0
			fpsLastTime = now
		}
	}

	/* Loop */
	draw := func() {
		updateFps()
		// Hintergrund + Nebel
		lib3d.Background(40, 40, 40)
		lib3d.SetFog(100, 400, 0.25, 0.25, 0.25, 1)

		// Matrizen: Blick (View) + Perspektive (Projektion)
		view := lib3d.LookAtMatrix(camPos, camTarget, camUp)
		proj := lib3d.PerspectiveMatrix(fovY, float64(lib3d.GetWidth())/float64(lib3d.GetHeight()), zNear, zFar)
		lib3d.SetProjection(proj)
		lib3d.SetView(view)

		// Lichtrichtung der „Sonne“ in den Kameraraum drehen
		camLight := sunDir.TransformDir(view)
		lib3d.SetLightDirection(camLight.X, camLight.Y, camLight.Z)

		// OPTIONAL (Animation demonstrieren): Kugel rotieren lassen
		ball.RotY += 0.01

		// Objekte zeichnen (Bodengitter zuerst für korrekte Tiefe)
		lib3d.DrawBody(grid)
		lib3d.DrawBody(ball)
	}

	/* Start */
	runtime.LockOSThread()
	if err := lib3d.Init(screenW, screenH); err != nil {
		log.Fatal(err)
	}
	lib3d.StartAnimation(draw)
}
