package main

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"time"

	"lib3dgo/lib3d"
)

// ====================================================================
// START
// ====================================================================
func main() {
	const (
		screenW   = 1600
		screenH   = 1000
		fovY      = 1.2
		zNear     = 0.1
		zFar      = 1000
		coneLines = 20
		camHeight = 140
	)

	var (
		camTarget  = lib3d.Vec3{X: 0, Y: 0, Z: 0}
		camUp      = lib3d.Vec3{X: 0, Y: 1, Z: 0}
		camRadius  = math.Sqrt(40*40 + 180*180)
		timeAccum  float64
		coneRotY   float64
		frameCount int
		fpsLast    float64
		boxMesh    = lib3d.CreateBoxSolid(100, 80, 60)
		pyrMesh    = lib3d.CreatePyramidSolid(90, 120)
	)

	var bodies = []*lib3d.Body{
		lib3d.NewBody(boxMesh, 150, 0, 50, &lib3d.BodyConfig{Color: "#ff0000", LineWidth: 2}),
		lib3d.NewBody(boxMesh, 0, 0, 100, &lib3d.BodyConfig{Color: "#00ffff", LineWidth: 2}),
		lib3d.NewBody(boxMesh, -150, 0, -100, &lib3d.BodyConfig{Color: "#ff0000", LineWidth: 2}),
		lib3d.NewBody(boxMesh, -200, 0, 30, &lib3d.BodyConfig{Color: "#00ffff", LineWidth: 2, RotY: math.Pi / 2}),
		lib3d.NewBody(pyrMesh, 100, 0, -100, nil), // Default: weiß, LineWidth 1
	}

	/* Loop */
	draw := func() {
		timeAccum += 0.02
		frameCount++
		now := float64(time.Now().UnixNano()) / 1e9
		if now-fpsLast >= 2.0 {
			fmt.Printf("FPS: %.1f\n", float64(frameCount)/(now-fpsLast))
			frameCount = 0
			fpsLast = now
		}

		camAngle := timeAccum * 0.15
		camPos := lib3d.Vec3{
			X: math.Sin(camAngle) * camRadius,
			Y: camHeight,
			Z: math.Cos(camAngle) * camRadius,
		}
		view := lib3d.LookAtMatrix(camPos, camTarget, camUp)
		proj := lib3d.PerspectiveMatrix(fovY, float64(lib3d.GetWidth())/float64(lib3d.GetHeight()), zNear, zFar)
		sunDir := lib3d.Vec3{X: 1, Y: 0, Z: 0}
		camLight := sunDir.TransformDir(view)
		lib3d.SetLightDirection(camLight.X, camLight.Y, camLight.Z)

		lib3d.SetProjection(proj)
		lib3d.SetView(view)
		lib3d.Background(40, 40, 40)

		// Alle Bodies zeichnen
		for _, b := range bodies {
			lib3d.DrawBody(b)
		}

		// Lichtkegel rotieren
		coneRotY += 0.01
		coneRot := lib3d.RotateMatrix(0, coneRotY, 0)

		// Clipping-Ebenen der Bodies (pro Frame neu)
		allPlanes := make([][]lib3d.Plane, len(bodies))
		for i, b := range bodies {
			allPlanes[i] = b.GetFacePlanes()
		}

		// --- Rays ---
		apex := lib3d.Vec3{}
		coneLen := 600.0
		coneAngle := math.Pi / 60
		coneR := coneLen * math.Sin(coneAngle)
		coneZ := coneLen * math.Cos(coneAngle)

		for i := 0; i < coneLines; i++ {
			a := (2.0 * math.Pi * float64(i)) / coneLines
			end := lib3d.Vec3{
				X: apex.X + math.Cos(a)*coneR,
				Y: apex.Y + math.Sin(a)*coneR,
				Z: apex.Z + coneZ,
			}

			rotatedEnd := lib3d.RotateAround(end, apex, coneRot)
			endpoint := rotatedEnd
			maxDist := rotatedEnd.Sub(apex).SquaredLength()

			for _, planes := range allPlanes {
				for _, face := range planes {
					if hit, ok := face.IntersectLine(apex, rotatedEnd); ok {
						dist := hit.Sub(apex).SquaredLength()
						if dist < maxDist {
							endpoint = hit
							maxDist = dist
						}
					}
				}
			}

			lib3d.DrawLine(
				apex.X, apex.Y, apex.Z, endpoint.X, endpoint.Y, endpoint.Z,
				lib3d.PrimitiveStyle{Stroke: "#ff8800", LineWidth: 1},
			)
			lib3d.DrawPoint(
				endpoint.X, endpoint.Y, endpoint.Z,
				lib3d.PrimitiveStyle{Stroke: "#ff0000", PointSize: 5},
			)
		}
	}

	/* Start */
	runtime.LockOSThread()
	if err := lib3d.Init(screenW, screenH); err != nil {
		log.Fatal(err)
	}
	lib3d.SetFog(100, 600, 0.25, 0.25, 0.25, 1)
	lib3d.StartAnimation(draw)
}
