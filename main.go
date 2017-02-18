package main

import (
    "fmt"
    "runtime"
    "strings"
    "github.com/go-gl/gl/v3.3-core/gl"
    "github.com/go-gl/glfw/v3.2/glfw"
    _"github.com/go-gl/mathgl/mgl32"
)

var (
    width int
    height int
    vertexArrayID uint32
    vertexBufferID uint32
    shaderProgramID uint32
    textureID uint32
)

func init() {
    runtime.LockOSThread()
}

func main() {

    width = 640
    height = 480

    err := glfw.Init()
    if err != nil {
        panic(err)
    }
    defer glfw.Terminate()

    glfw.WindowHint(glfw.Resizable, glfw.False)
    glfw.WindowHint(glfw.ContextVersionMajor, 3)
    glfw.WindowHint(glfw.ContextVersionMinor, 3)
    glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
    glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

    window, err := glfw.CreateWindow(width, height, "goray", nil, nil)
    if err != nil {
        panic(err)
    }

    window.MakeContextCurrent()

    if err := gl.Init(); err != nil {
        panic(err)
    }

    quit := make(chan bool, 1)

    var mainLoop func()
    mainLoop = func() {
        gl.ClearColor(0.0, 0.0, 0.4, 0.0)
        gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

        gl.UseProgram(shaderProgramID)
        gl.BindVertexArray(vertexArrayID)
        gl.ActiveTexture(gl.TEXTURE0)
        gl.BindTexture(gl.TEXTURE_2D, textureID)
        gl.DrawArrays(gl.TRIANGLES, 0, 6)

        window.SwapBuffers()
        glfw.PollEvents()
        if window.ShouldClose() {
            quit <- true
        }
        go do(mainLoop)
    }

    go do(setupScene)

    go do(mainLoop)

    loop:
    for f := range mainfunc {
        f()
        select {
            case shouldQuit := <-quit:
                if shouldQuit {
                    break loop
                }
            default:
                continue
        }
    }
}

var mainfunc = make(chan func(), 100)

func do(f func()) {
    done := make(chan bool, 1)
    mainfunc <- func() {
        f()
        done <- true
    }
    <-done
}

func setupScene() {
    gl.GenVertexArrays(1, &vertexArrayID)
    gl.BindVertexArray(vertexArrayID)

    gl.GenBuffers(1, &vertexBufferID)
    gl.BindBuffer(gl.ARRAY_BUFFER, vertexBufferID)

    vertexBufferData := []float32{
        -1.0, -1.0, 0.0, 1.0, 0.0,
         1.0, -1.0, 0.0, 0.0, 0.0,
        -1.0,  1.0, 0.0, 1.0, 1.0,
         1.0, -1.0, 0.0, 0.0, 0.0,
         1.0,  1.0, 1.0, 0.0, 1.0,
        -1.0,  1.0, 1.0, 1.0, 1.0,
    }

    gl.BufferData(gl.ARRAY_BUFFER, len(vertexBufferData)*4, gl.Ptr(vertexBufferData), gl.STATIC_DRAW)

    gl.EnableVertexAttribArray(0);
    gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

    gl.EnableVertexAttribArray(1);
    gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))

    var err error;
    shaderProgramID, err = newProgram(vertexShader, fragmentShader)
    if err != nil {
        panic(err)
    }

    gl.GenTextures(1, &textureID)
    gl.ActiveTexture(gl.TEXTURE0)

    nx := 20
    ny := 20

    tex := make([]uint8, nx*ny*3)

    for j := 0; j < ny; j++ {
        for i := 0; i < nx; i++ {
            r := float32(i)/float32(nx);
            g := float32(j)/float32(ny);
            b := 0.2;
            tex[i*j+0] = uint8(255.99*r)
            tex[i*j+1] = uint8(255.99*g)
            tex[i*j+2] = uint8(255.99*b)
        }
    }


    gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
    setTexture(nx, ny, tex)
}

func setTexture(w int, h int, data []uint8) {
    gl.BindTexture(gl.TEXTURE_2D, textureID)

    gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
    gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

    gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
    gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

    gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB, int32(w), int32(h), 0, gl.RGB, gl.UNSIGNED_BYTE, gl.Ptr(data))

    gl.BindTexture(gl.TEXTURE_2D, 0)
}

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

var vertexShader = `
#version 330 core
layout(location = 0) in vec3 vPos;
layout(location = 1) in vec2 vUV;

out vec2 uv;

void main() {
    gl_Position.xyzw = vec4(vPos, 1.0);
    uv = vUV;
}
` + "\x00"

var fragmentShader = `
#version 330 core
in vec2 uv;
out vec3 color;
uniform sampler2D tex;
void main() {
    color = texture(tex, uv).rgb;
}
` + "\x00"