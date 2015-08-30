// Copyright 2015, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"errors"
	"fmt"
	gl "github.com/go-gl/gl/v3.3-core/gl"
	glfw "github.com/go-gl/glfw/v3.1/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"
	"runtime"
	"strings"
	"time"
)

var (
	// UnlitColorVertShader is a basic color vertex shader
	UnlitColorVertShader = `#version 330
	uniform mat4 MVP_MATRIX;
	uniform vec4 DIFFUSE_COLOR;
  in vec3 VERTEX_POSITION;
	out vec4 vs_diffuse;

  void main()
  {
		vs_diffuse = DIFFUSE_COLOR;
		gl_Position = MVP_MATRIX * vec4(VERTEX_POSITION, 1.0);
  }`

	// UnlitColorFragShader is a basic color fragment shader
	UnlitColorFragShader = `#version 330
	in vec4 vs_diffuse;
  out vec4 colourOut;
  void main()
  {
    colourOut = vs_diffuse;
  }`
)

// GLFW event handling must run on the main OS thread
func init() {
	runtime.LockOSThread()
}

type RenderLoopCallback func(delta float64)

// ExampleApp is a type representing the example application and holds on
// to related data like OpenGL windows.
type ExampleApp struct {
	// MainWindow is the main OpenGL window for the application
	MainWindow *glfw.Window

	// Width is how wide the app window is
	Width int

	// Height is how tall the app window is
	Height int

	// CameraPos is the position of the camera in world space
	CameraPos mgl.Vec3

	// CameraRotation is a quaternion representing the direction
	// the camera is looking.
	CameraRotation mgl.Quat

	// OnUpdate is called just prior to OnRender and can be used to update
	// the application data.
	OnUpdate RenderLoopCallback

	// OnRender is called at the end of the render loop and is meant to be
	// the spot where the application renders the objects to OpenGL.
	OnRender RenderLoopCallback
}

// NewApp returns a new ExampleApp object to control the display of the example app.
func NewApp() *ExampleApp {
	app := new(ExampleApp)
	app.CameraRotation = mgl.QuatIdent()
	return app
}

// InitGraphics creates an OpenGL window and initializes the required graphics libraries.
// It will either succeed or panic.
func (app *ExampleApp) InitGraphics(title string, w int, h int) {
	err := glfw.Init()
	if err != nil {
		panic("Can't init glfw! " + err.Error())
	}

	// request a OpenGL 3.3 core context
	glfw.WindowHint(glfw.Samples, 0)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	// do the actual window creation
	app.MainWindow, err = glfw.CreateWindow(w, h, title, nil, nil)
	if err != nil {
		panic("Failed to create the main window! " + err.Error())
	}
	app.MainWindow.MakeContextCurrent()
	glfw.SwapInterval(0)

	// make sure that all of the GL functions are initialized
	err = gl.Init()
	if err != nil {
		panic("Failed to initialize GL! " + err.Error())
	}

	// set the app window dimensions
	app.Width = w
	app.Height = h
}

// Terminate closes the OpenGL window and unloads the graphics libraries.
func (app *ExampleApp) Terminate() {
	app.MainWindow.SetShouldClose(true)
	glfw.Terminate()
}

// SetKeyCallback sets a key handler for the main window.
func (app *ExampleApp) SetKeyCallback(cb glfw.KeyCallback) {
	app.MainWindow.SetKeyCallback(cb)
}

var (
	// keeps track of the start of the last render loop
	lastRenderTime time.Time
)

// RenderLoop is the main render loop for the application
func (app *ExampleApp) RenderLoop() {
	lastRenderTime = time.Now()

	for !app.MainWindow.ShouldClose() {
		// get the time delta
		loopTime := time.Now()
		deltaNano := loopTime.Sub(lastRenderTime).Nanoseconds()
		deltaF := float64(deltaNano) * (1.0 / float64(time.Second))

		// call the Update callback
		if app.OnUpdate != nil {
			app.OnUpdate(deltaF)
		}

		// call the Render callback
		if app.OnRender != nil {
			app.OnRender(deltaF)
		}

		// draw the screen and get any input
		app.MainWindow.SwapBuffers()
		glfw.PollEvents()

		// update the last render time
		lastRenderTime = loopTime
	}
}

// Renderable is an object that can be drawn in the render loop
type Renderable struct {
	// Shader is the shader program to use to draw the renderable
	Shader uint32

	// Tex0 is the first texture to be bound to the shader
	Tex0 uint32

	// Color is a material color for the object passed to the shader when drawn.
	Color mgl.Vec4

	// Vao is the VAO object used to draw the object
	Vao uint32

	// VertVBO is the VBO that holds the vertex data
	VertVBO uint32

	// UvVBO is the VBO that holds the UV data
	UvVBO uint32

	// NormsVBO is the VBO that hold the normal data
	NormsVBO uint32

	// ElementsVBO is the VBO
	ElementsVBO uint32

	// FaceCount is the number of faces to draw for the object
	FaceCount int

	// Scale represents how to scale the object when drawing
	Scale mgl.Vec3

	// Location positions the object in world space
	Location mgl.Vec3

	// Rotation is the rotation of the object in world space
	Rotation      mgl.Quat

	// LocalRotation is rotation applied to the object in local space
	LocalRotation mgl.Quat
}

// NewRenderable creates a new Renderable object.
func NewRenderable() *Renderable {
	r := new(Renderable)
	r.Scale = mgl.Vec3{1.0, 1.0, 1.0}
	return r
}

// GetTransformMat4 creates a transform matrix: scale * transform
func (r *Renderable) GetTransformMat4() mgl.Mat4 {
	scaleMat := mgl.Scale3D(r.Scale[0], r.Scale[1], r.Scale[2])
	transMat := mgl.Translate3D(r.Location[0], r.Location[1], r.Location[2])
	localRotMat := r.LocalRotation.Mat4()
	rotMat := r.Rotation.Mat4()
	modelTransform := rotMat.Mul4(transMat).Mul4(localRotMat).Mul4(scaleMat)
	return modelTransform
}

func (r *Renderable) Draw(perspective mgl.Mat4, view mgl.Mat4) {
	gl.UseProgram(r.Shader)
	gl.BindVertexArray(r.Vao)

	model := r.GetTransformMat4()

	var mvp mgl.Mat4
	shaderMvp := getUniformLocation(r.Shader, "MVP_MATRIX")
	if shaderMvp >= 0 {
		mvp = perspective.Mul4(view).Mul4(model)
		gl.UniformMatrix4fv(shaderMvp, 1, false, &mvp[0])
	}

	shaderTex0 := getUniformLocation(r.Shader, "DIFFUSE_TEX")
	if shaderTex0 >= 0 {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, r.Tex0)
		gl.Uniform1i(shaderTex0, 0)
	}

	shaderColor := getUniformLocation(r.Shader, "DIFFUSE_COLOR")
	if shaderColor >= 0 {
		gl.Uniform4f(shaderColor, r.Color[0], r.Color[1], r.Color[2], r.Color[3])
	}

	shaderPosition := getAttribLocation(r.Shader, "VERTEX_POSITION")
	if shaderPosition >= 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, r.VertVBO)
		gl.EnableVertexAttribArray(uint32(shaderPosition))
		gl.VertexAttribPointer(uint32(shaderPosition), 3, gl.FLOAT, false, 0, gl.PtrOffset(0))
	}

	shaderVertUv := getAttribLocation(r.Shader, "VERTEX_UV_0")
	if shaderVertUv >= 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, r.UvVBO)
		gl.EnableVertexAttribArray(uint32(shaderVertUv))
		gl.VertexAttribPointer(uint32(shaderVertUv), 2, gl.FLOAT, false, 0, gl.PtrOffset(0))
	}

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, r.ElementsVBO)
	gl.DrawElements(gl.TRIANGLES, int32(r.FaceCount*3), gl.UNSIGNED_INT, gl.PtrOffset(0))
	gl.BindVertexArray(0)
}

// setup a cache for the uniform and attribute getter functions
var (
	uniCache  = make(map[string]int32)
	attrCache = make(map[string]int32)
)

func getUniformLocation(prog uint32, name string) int32 {
	// attempt to get it from the cache first
	ul, found := uniCache[name]
	if found {
		return ul
	}

	// pull the location from the shader and cache it
	uniGLName := name + "\x00"
	ul = gl.GetUniformLocation(prog, gl.Str(uniGLName))

	// cache even if it returns -1 so that it doesn't repeatedly check
	uniCache[name] = ul
	return ul
}

func getAttribLocation(prog uint32, name string) int32 {
	// attempt to get it from the cache first
	al, found := attrCache[name]
	if found {
		return al
	}

	// pull the location from the shader and cache it
	attrGLName := name + "\x00"
	al = gl.GetAttribLocation(prog, gl.Str(attrGLName))

	// cache even if it returns -1 so that it doesn't repeatedly check
	attrCache[name] = al
	return al
}

// LoadShaderProgram loads shader objects and then attaches them to a program
func LoadShaderProgram(vertShader, fragShader string) (uint32, error) {
	// create the program
	prog := gl.CreateProgram()

	// create the vertex shader
	vs := gl.CreateShader(gl.VERTEX_SHADER)
	cVertShader := gl.Str(vertShader + "\x00")
	gl.ShaderSource(vs, 1, &cVertShader, nil)
	gl.CompileShader(vs)

	var status int32
	gl.GetShaderiv(vs, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(vs, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(vs, logLength, nil, gl.Str(log))

		err := fmt.Sprintf("Failed to compile the vertex shader!\n%s", log)
		fmt.Println(err)
		return 0, errors.New(err)
	}
	defer gl.DeleteShader(vs)

	// create the fragment shader
	fs := gl.CreateShader(gl.FRAGMENT_SHADER)
	cFragShader := gl.Str(fragShader + "\x00")
	gl.ShaderSource(fs, 1, &cFragShader, nil)
	gl.CompileShader(fs)

	gl.GetShaderiv(fs, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(fs, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(fs, logLength, nil, gl.Str(log))

		err := fmt.Sprintf("Failed to compile the fragment shader!\n%s", log)
		fmt.Println(err)
		return 0, errors.New(err)
	}
	defer gl.DeleteShader(fs)

	// attach the shaders to the program and link
	// attach the shaders to the program and link
	gl.AttachShader(prog, vs)
	gl.AttachShader(prog, fs)
	gl.LinkProgram(prog)

	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(prog, logLength, nil, gl.Str(log))

		error := fmt.Sprintf("Failed to link the program!\n%s", log)
		fmt.Println(error)
		return 0, errors.New(error)
	}

	return prog, nil
}

// CreateCube makes a new Renderable object with the specified dimensions for the cube.
func CreateCube(xmin, ymin, zmin, xmax, ymax, zmax float32) *Renderable {
	/* Cube vertices are layed out like this:

	  +--------+           6          5
	/ |       /|
	+--------+ |        1          0        +Y
	| |      | |                            |___ +X
	| +------|-+           7          4    /
	|/       |/                           +Z
	+--------+          2          3

	*/

	verts := [...]float32{
		xmax, ymax, zmax, xmin, ymax, zmax, xmin, ymin, zmax, xmax, ymin, zmax, // v0,v1,v2,v3 (front)
		xmax, ymax, zmin, xmax, ymax, zmax, xmax, ymin, zmax, xmax, ymin, zmin, // v5,v0,v3,v4 (right)
		xmax, ymax, zmin, xmin, ymax, zmin, xmin, ymax, zmax, xmax, ymax, zmax, // v5,v6,v1,v0 (top)
		xmin, ymax, zmax, xmin, ymax, zmin, xmin, ymin, zmin, xmin, ymin, zmax, // v1,v6,v7,v2 (left)
		xmax, ymin, zmax, xmin, ymin, zmax, xmin, ymin, zmin, xmax, ymin, zmin, // v3,v2,v7,v4 (bottom)
		xmin, ymax, zmin, xmax, ymax, zmin, xmax, ymin, zmin, xmin, ymin, zmin, // v6,v5,v4,v7 (back)
	}
	indexes := [...]uint32{
		0, 1, 2, 2, 3, 0,
		4, 5, 6, 6, 7, 4,
		8, 9, 10, 10, 11, 8,
		12, 13, 14, 14, 15, 12,
		16, 17, 18, 18, 19, 16,
		20, 21, 22, 22, 23, 20,
	}
	uvs := [...]float32{
		1.0, 1.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0,
		1.0, 1.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0,
		1.0, 1.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0,
		1.0, 1.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0,
		1.0, 1.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0,
		1.0, 1.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0,
	}
	normals := [...]float32{
		0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, // v0,v1,v2,v3 (front)
		1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, // v5,v0,v3,v4 (right)
		0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, // v5,v6,v1,v0 (top)
		-1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, // v1,v6,v7,v2 (left)
		0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, // v3,v2,v7,v4 (bottom)
		0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, // v6,v5,v4,v7 (back)
	}

	r := NewRenderable()
	gl.GenVertexArrays(1, &r.Vao)
	r.FaceCount = 12

	const floatSize = 4
	const uintSize = 4

	// create a VBO to hold the vertex data
	gl.GenBuffers(1, &r.VertVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.VertVBO)
	gl.BufferData(gl.ARRAY_BUFFER, floatSize*len(verts), gl.Ptr(&verts[0]), gl.STATIC_DRAW)

	// create a VBO to hold the uv data
	gl.GenBuffers(1, &r.UvVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.UvVBO)
	gl.BufferData(gl.ARRAY_BUFFER, floatSize*len(uvs), gl.Ptr(&uvs[0]), gl.STATIC_DRAW)

	// create a VBO to hold the normals data
	gl.GenBuffers(1, &r.NormsVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.NormsVBO)
	gl.BufferData(gl.ARRAY_BUFFER, floatSize*len(normals), gl.Ptr(&normals[0]), gl.STATIC_DRAW)

	// create a VBO to hold the face indexes
	gl.GenBuffers(1, &r.ElementsVBO)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, r.ElementsVBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, uintSize*len(indexes), gl.Ptr(&indexes[0]), gl.STATIC_DRAW)

	return r
}
