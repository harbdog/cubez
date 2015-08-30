// Copyright 2015, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	gl "github.com/go-gl/gl/v3.3-core/gl"
	glfw "github.com/go-gl/glfw/v3.1/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/tbogdala/cubez"
	m "github.com/tbogdala/cubez/math"
)

var (
	app *ExampleApp

  cube *Renderable
	cubeCollider *cubez.CollisionCube

	colorShader uint32
)

// update object locations
func updateObjects(delta float64) {
	// for now there's only one box to update
	cubeCollider.Body.Integrate(m.Real(delta))
	cubeCollider.CalculateDerivedData()

	// for now we hack in the position and rotation
	// of the collider into the renderable
	cube.Location = mgl.Vec3{
		float32(cubeCollider.Body.Position[0]),
		float32(cubeCollider.Body.Position[1]),
		float32(cubeCollider.Body.Position[2]),
		}
	cube.LocalRotation = mgl.Quat{
		float32(cubeCollider.Body.Orientation[0]),
		mgl.Vec3{
			float32(cubeCollider.Body.Orientation[1]),
			float32(cubeCollider.Body.Orientation[2]),
			float32(cubeCollider.Body.Orientation[3]),
		},
		}
}

// see if any of the rigid bodys contact
func generateContacts(delta float64) (bool, []*cubez.Contact) {
	// create the ground plane
	groundPlane := cubez.NewCollisionPlane(m.Vector3{0.0, 1.0, 0.0}, 0.0)

	// see if we have a collision with the ground
	return cubeCollider.CheckAgainstHalfSpace(groundPlane, nil)
}

func updateCallback(delta float64)  {
	updateObjects(delta)
	foundContacts, contacts := generateContacts(delta)
	if foundContacts {
		cubez.ResolveContacts(len(contacts)*8, contacts, m.Real(delta))
	}
}

func renderCallback(delta float64)  {
	gl.Viewport(0, 0, int32(app.Width), int32(app.Height))
	gl.ClearColor(0.05, 0.05, 0.05, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// make the projection and view matrixes
	projection := mgl.Perspective(mgl.DegToRad(60.0), float32(app.Width)/float32(app.Height), 1.0, 200.0)
	view := app.CameraRotation.Mat4()
	view = view.Mul4(mgl.Translate3D(-app.CameraPos[0], -app.CameraPos[1], -app.CameraPos[2]))

	cube.Draw(projection, view)

}

func main() {
	app = NewApp()
	app.InitGraphics("Ballistic", 800, 600)
	app.SetKeyCallback(keyCallback)
	app.OnRender = renderCallback
	app.OnUpdate = updateCallback
	defer app.Terminate()

	// compile the shaders
	var err error
	colorShader, err = LoadShaderProgram(UnlitColorVertShader, UnlitColorFragShader)
	if err != nil {
		panic("Failed to compile the vertex shader! " + err.Error())
	}

  // create a test cube to render
  cube = CreateCube(-0.5, -0.5, -0.5, 0.5, 0.5, 0.5)
	cube.Shader = colorShader
	cube.Color = mgl.Vec4{1.0, 0.0, 0.0, 1.0}


	// create the collision box for the the cube
	cubeCollider = cubez.NewCollisionCube(nil, m.Vector3{0.5, 0.5, 0.5})
	cubeCollider.Body.Position = m.Vector3{0.0, 4.0, 0.0}
	cubeCollider.Body.SetMass(10.0)
	cubeCollider.Body.CalculateDerivedData()
	cubeCollider.CalculateDerivedData()


	// setup the camera
	app.CameraPos = mgl.Vec3{0.0, 0.0, 5.0}

	gl.Enable(gl.DEPTH_TEST)
	app.RenderLoop()
}

func keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// Key W == close app
	if key == glfw.KeyEscape && action == glfw.Press {
		w.SetShouldClose(true)
	}
}
