package lib3d

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// =================================================================
// INTERNER GLSL-CODE  (Port von lib-wgl.ts)
//
// Unterschied zum WebGL-Original: Desktop OpenGL 3.3 Core. dFdx/dFdy sind
// im Core-Profil immer verfügbar (keine OES_standard_derivatives-Extension),
// und es gibt keine Präzisions-Qualifier (highp/mediump) mehr.
// =================================================================

// Vertex-Shader:
// 3D-Transformation über ModelView- und Projection-Matrix.
// aPos = 3D-Punkt in Objektkoordinaten.
const vertSrc = `#version 330 core
layout(location = 0) in vec3 aPos;
uniform mat4 uView;
uniform mat4 uModel;
uniform mat4 uProjection;
uniform float uPointSize;

out vec3 vCamPos;

void main() {
    vec4 worldPos = uModel * vec4(aPos, 1.0);
    vec4 camPos = uView * worldPos;
    vCamPos = camPos.xyz;
    gl_Position = uProjection * camPos;
    gl_PointSize = uPointSize;
}
`

// Fragment-Shader:
// uMode  0 = flat      – einfache Farbe uColor
// uMode  1 = gradient  – radialer Verlauf (vom Zentrum in Kamera-Koordinaten)
// uMode  2 = pulse     – flat mit sinusförmiger Helligkeitspulsation
// uLighted == 1 → Flat-Shading (Beleuchtung).
//
// Die Facetten-Normale wird pro Pixel aus den Kamera-Raum-Ableitungen
// (dFdx/dFdy) rekonstruiert → kein Normalen-Attribut nötig. cross(dFdy, dFdx)
// sorgt dafür, dass die rekonstruierte Normale zur Kamera zeigt
// (linkshändiger View-Raum, Kamera blickt in +Z).
const fragSrc = `#version 330 core

in vec3 vCamPos;
out vec4 fragColor;

uniform int   uMode;
uniform vec4  uColor;
uniform vec4  uColor2;
uniform float uTime;
uniform vec3  uShapeCenter;   // Zentrum in Kamera-Koordinaten
uniform float uShapeRadius;
uniform float uFogNear;
uniform float uFogFar;
uniform vec4  uFogColor;
uniform int   uLighted;       // 1 = gefüllte Fläche → beleuchten
uniform vec3  uLightDir;      // Lichtrichtung in Kamera-Koordinaten

void main() {
  vec3 base = uColor.rgb;

  if (uMode == 0) {
    // flat – Basis bleibt uColor
  } else if (uMode == 1) {
    float d = distance(vCamPos, uShapeCenter);
    float t = clamp(d / max(uShapeRadius, 1.0), 0.0, 1.0);
    base = mix(uColor, uColor2, t).rgb;
  } else if (uMode == 2) {
    float brightness = 0.6 + 0.4 * sin(uTime * 3.0);
    base = uColor.rgb * brightness;
  }

  if (uLighted == 1) {
    vec3 n = normalize(cross(dFdy(vCamPos), dFdx(vCamPos)));
    float diff = max(dot(n, normalize(uLightDir)), 0.0);
    base *= 0.25 + 0.75 * diff;
  }

  // Nebel (Tiefen-basiert im Kamera-Raum)
  float depth = vCamPos.z;
  float fog_t = clamp((depth - uFogNear) / (uFogFar - uFogNear), 0.0, 1.0);
  fragColor = mix(vec4(base, uColor.a), uFogColor, fog_t);
}
`

// compileShader kompiliert einen einzelnen GLSL-Shader.
func compileShader(xtype uint32, src string) (uint32, error) {
	shader := gl.CreateShader(xtype)
	csrc, free := gl.Strs(src + "\x00")
	gl.ShaderSource(shader, 1, csrc, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		return 0, fmt.Errorf("shader-fehler: %s", getShaderInfoLog(shader))
	}
	return shader, nil
}

// createProgram verknüpft Vertex- und Fragment-Shader zu einem GL-Programm.
func createProgram(vs, fs string) (uint32, error) {
	vert, err := compileShader(gl.VERTEX_SHADER, vs)
	if err != nil {
		return 0, err
	}
	frag, err := compileShader(gl.FRAGMENT_SHADER, fs)
	if err != nil {
		return 0, err
	}

	p := gl.CreateProgram()
	gl.AttachShader(p, vert)
	gl.AttachShader(p, frag)
	gl.LinkProgram(p)

	var status int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		return 0, fmt.Errorf("programm-fehler: %s", getProgramInfoLog(p))
	}
	return p, nil
}

// getShaderInfoLog liest das Info-Log eines Shaders.
func getShaderInfoLog(shader uint32) string {
	var logLength int32
	gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
	if logLength <= 0 {
		return ""
	}
	buf := make([]byte, logLength)
	gl.GetShaderInfoLog(shader, logLength, nil, &buf[0])
	return string(buf)
}

// getProgramInfoLog liest das Info-Log eines Programms.
func getProgramInfoLog(program uint32) string {
	var logLength int32
	gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
	if logLength <= 0 {
		return ""
	}
	buf := make([]byte, logLength)
	gl.GetProgramInfoLog(program, logLength, nil, &buf[0])
	return string(buf)
}
