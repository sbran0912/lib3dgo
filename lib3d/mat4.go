package lib3d

import "math"

// ====================================================================
// 4x4-MATRIZEN  (Port von lib-3d.ts: Matrix-Helfer)
// ====================================================================

// TranslateMatrix erzeugt eine 4x4-Translationsmatrix.
func TranslateMatrix(dx, dy, dz float64) Matrix4x4 {
	return Matrix4x4{
		{1, 0, 0, dx},
		{0, 1, 0, dy},
		{0, 0, 1, dz},
		{0, 0, 0, 1},
	}
}

// RotateMatrix erzeugt eine kombinierte 4x4-Rotationsmatrix (Ry * Rx) * Rz.
func RotateMatrix(ax, ay, az float64) Matrix4x4 {
	rx := Matrix4x4{
		{1, 0, 0, 0},
		{0, math.Cos(ax), -math.Sin(ax), 0},
		{0, math.Sin(ax), math.Cos(ax), 0},
		{0, 0, 0, 1},
	}
	ry := Matrix4x4{
		{math.Cos(ay), 0, math.Sin(ay), 0},
		{0, 1, 0, 0},
		{-math.Sin(ay), 0, math.Cos(ay), 0},
		{0, 0, 0, 1},
	}
	rz := Matrix4x4{
		{math.Cos(az), -math.Sin(az), 0, 0},
		{math.Sin(az), math.Cos(az), 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
	return MultMatrix(rz, MultMatrix(ry, rx))
}

// MultMatrix multipliziert zwei 4x4-Matrizen (a * b).
func MultMatrix(a, b Matrix4x4) Matrix4x4 {
	var result Matrix4x4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return result
}

// IdentityMatrix liefert die 4x4-Identitätsmatrix.
func IdentityMatrix() Matrix4x4 {
	return Matrix4x4{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}

// ViewWorldMatrix kombiniert View- und World-Matrix (view × world).
func ViewWorldMatrix(view, world Matrix4x4) Matrix4x4 {
	return MultMatrix(view, world)
}

// LookAtMatrix erzeugt eine 4x4-View-Matrix, die Weltkoordinaten in
// Kamerakoordinaten transformiert. Die Kamera blickt in +Z-Richtung
// (kompatibel mit PerspectiveMatrix).
func LookAtMatrix(cameraPos, target, up Vec3) Matrix4x4 {
	forward := target.Sub(cameraPos).Normalize()
	right := forward.Cross(up).Normalize()
	realUp := right.Cross(forward)

	return Matrix4x4{
		{right.X, right.Y, right.Z, -right.Dot(cameraPos)},
		{realUp.X, realUp.Y, realUp.Z, -realUp.Dot(cameraPos)},
		{forward.X, forward.Y, forward.Z, -forward.Dot(cameraPos)},
		{0, 0, 0, 1},
	}
}

// WorldToCamera transformiert einen Weltpunkt in Kamerakoordinaten
// (view × world).
func WorldToCamera(point Vec3, view, world Matrix4x4) Vec3 {
	return point.Transform(MultMatrix(view, world))
}

// RotateAround rotiert einen Punkt um ein Pivot (Drehpunkt).
func RotateAround(point, pivot Vec3, rotation Matrix4x4) Vec3 {
	rel := point.Sub(pivot)
	rotated := rel.Transform(rotation)
	return rotated.Add(pivot)
}

// PerspectiveMatrix erzeugt eine perspektivische Projektionsmatrix
// (WebGL-kompatibel). KONVENTION: Kamera blickt in +Z-Richtung.
func PerspectiveMatrix(fovY, aspect, near, far float64) Matrix4x4 {
	f := 1.0 / math.Tan(fovY/2)
	return Matrix4x4{
		{f / aspect, 0, 0, 0},
		{0, f, 0, 0},
		{0, 0, (far + near) / (far - near), -2 * near * far / (far - near)},
		{0, 0, 1, 0},
	}
}
