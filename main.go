package main

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"math"
	"os"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func init() {
	// This is needed to arrange that main() runs on main thread.
	// See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()
}

func main() {
	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(640, 480, "Testing", nil, nil)

	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()

	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	program, err := newProgramFile("basic")

	triangleVAO := getTriangleMesh(program)

	//previousTime := 0.0
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Update
		time := glfw.GetTime()
		//deltaTime := time - previousTime
		//previousTime = time

		// Configure global settings
		gl.Enable(gl.DEPTH_TEST)
		gl.DepthFunc(gl.LESS)
		gl.ClearColor(0.0, 0.3, 1.0, 1.0)

		gl.UseProgram(program)

		windowWidth, windowHeight := glfw.GetCurrentContext().GetSize()

		projection := mgl32.Perspective(
			mgl32.DegToRad(45.0), float32(windowWidth)/float32(windowHeight), 0.1, 10.0,
		)
		projectionUniform := gl.GetUniformLocation(program, glString("projection"))
		gl.UniformMatrix4fv(projectionUniform, 1, false, &projection[0])

		camera := mgl32.LookAtV(
			mgl32.Vec3{
				float32(math.Cos(time) * 2.0), 0, float32(math.Sin(time) * 2.0),
			},
			mgl32.Vec3{0, 0, 0},
			mgl32.Vec3{0, 1, 0},
		)
		cameraUniform := gl.GetUniformLocation(program, glString("camera"))
		gl.UniformMatrix4fv(cameraUniform, 1, false, &camera[0])

		model := mgl32.Ident4()
		modelUniform := gl.GetUniformLocation(program, glString("model"))
		gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])

		gl.BindVertexArray(triangleVAO)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)

		// Do OpenGL stuff.
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func getTriangleMesh(program uint32) uint32 {
	// Vertices with position and color
	vertices := []float32{
		// Positions        // Colors
		-0.5, -0.5, 0.0, 1.0, 0.0, 0.0, // left - red
		0.5, -0.5, 0.0, 0.0, 1.0, 0.0, // right - green
		0.0, 0.5, 0.0, 0.0, 0.0, 1.0, // top - blue
	}

	// Configure the vertex data
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	vertAttrib := uint32(gl.GetAttribLocation(program, glString("vert")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 3, gl.FLOAT, false, 6*4, 0)

	texCoordAttrib := uint32(gl.GetAttribLocation(program, glString("vertColor")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 3, gl.FLOAT, false, 6*4, 3*4)

	return vao
}

func glString(str string) *uint8 {
	return gl.Str(str + "\x00")
}

func newProgramFile(fileName string) (uint32, error) {
	content, err := os.ReadFile(fileName + ".glsl_vert")
	if err != nil {
		log.Fatal(err)
	}
	vertexShader := string(content)

	content, err = os.ReadFile(fileName + ".glsl_frag")
	if err != nil {
		log.Fatal(err)
	}
	fragmentShader := string(content)

	// Configure the vertex and fragment shaders
	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	return program, err
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

func newTexture(file string) (uint32, error) {
	imgFile, err := os.Open(file)
	if err != nil {
		return 0, fmt.Errorf("texture %q not found on disk: %v", file, err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return 0, err
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture, nil
}
