package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"sync"

	"github.com/paked/engi"
	"github.com/paked/engi/ecs"
)

type PongGame struct{}

var (
	basicFont *engi.Font
)

func (pong *PongGame) Preload() {
	engi.Files.AddFromDir("assets", true)
}

func (pong *PongGame) Setup(w *ecs.World) {
	engi.SetBg(0x2d3739)
	w.AddSystem(&engi.RenderSystem{})
	w.AddSystem(&engi.CollisionSystem{})
	w.AddSystem(&SpeedSystem{})
	w.AddSystem(&ControlSystem{})
	w.AddSystem(&BallSystem{})
	w.AddSystem(&ScoreSystem{})

	basicFont = (&engi.Font{URL: "Roboto-Regular.ttf", Size: 32, FG: color.NRGBA{255, 255, 255, 255}})
	if err := basicFont.CreatePreloaded(); err != nil {
		log.Fatalln("Could not load font:", err)
	}

	ball := ecs.NewEntity([]string{"RenderSystem", "CollisionSystem", "SpeedSystem", "BallSystem"})
	ballTexture := engi.Files.Image("ball.png")
	ballRender := engi.NewRenderComponent(ballTexture, engi.Point{2, 2}, "ball")
	ballSpace := &engi.SpaceComponent{engi.Point{(engi.Width() - ballTexture.Width()) / 2, (engi.Height() - ballTexture.Height()) / 2}, ballTexture.Width() * ballRender.Scale().X, ballTexture.Height() * ballRender.Scale().Y}
	ballCollision := &engi.CollisionComponent{Main: true, Solid: true}
	ballSpeed := &SpeedComponent{}
	ballSpeed.Point = engi.Point{300, 100}

	ball.AddComponent(ballRender)
	ball.AddComponent(ballSpace)
	ball.AddComponent(ballCollision)
	ball.AddComponent(ballSpeed)
	err := w.AddEntity(ball)
	if err != nil {
		log.Println(err)
	}

	score := ecs.NewEntity([]string{"RenderSystem", "ScoreSystem"})

	scoreRender := engi.NewRenderComponent(basicFont.Render(" "), engi.Point{1, 1}, "YOLO <3")

	scoreSpace := &engi.SpaceComponent{engi.Point{100, 100}, 100, 100}
	score.AddComponent(scoreRender)
	score.AddComponent(scoreSpace)
	err = w.AddEntity(score)
	if err != nil {
		log.Println(err)
	}

	schemes := []string{"WASD", ""}
	for i := 0; i < 2; i++ {
		paddle := ecs.NewEntity([]string{"RenderSystem", "CollisionSystem", "ControlSystem"})
		paddleTexture := engi.Files.Image("paddle.png")
		paddleRender := engi.NewRenderComponent(paddleTexture, engi.Point{2, 2}, "paddle")
		x := float32(0)
		if i != 0 {
			x = 800 - 16
		}

		paddleSpace := &engi.SpaceComponent{engi.Point{x, (engi.Height() - paddleTexture.Height()) / 2}, paddleRender.Scale().X * paddleTexture.Width(), paddleRender.Scale().Y * paddleTexture.Height()}
		paddleControl := &ControlComponent{schemes[i]}
		paddleCollision := &engi.CollisionComponent{Main: false, Solid: true}
		paddle.AddComponent(paddleRender)
		paddle.AddComponent(paddleSpace)
		paddle.AddComponent(paddleControl)
		paddle.AddComponent(paddleCollision)
		err = w.AddEntity(paddle)
		if err != nil {
			log.Println(err)
		}
	}
}

func (*PongGame) Hide()        {}
func (*PongGame) Show()        {}
func (*PongGame) Type() string { return "PongGame" }

type SpeedSystem struct {
	ecs.LinearSystem
}

func (*SpeedSystem) Type() string { return "SpeedSystem" }
func (*SpeedSystem) Pre()         {}
func (*SpeedSystem) Post()        {}

func (ms *SpeedSystem) New(*ecs.World) {
	engi.Mailbox.Listen("CollisionMessage", func(message engi.Message) {
		log.Println("collision")
		collision, isCollision := message.(engi.CollisionMessage)
		if isCollision {
			var speed *SpeedComponent
			if !collision.Entity.Component(&speed) {
				return
			}

			speed.X *= -1
		}
	})
}

func (ms *SpeedSystem) UpdateEntity(entity *ecs.Entity, dt float32) {
	var speed *SpeedComponent
	var space *engi.SpaceComponent
	if !entity.Component(&speed) || !entity.Component(&space) {
		return
	}
	space.Position.X += speed.X * dt
	space.Position.Y += speed.Y * dt
}

func (ms *SpeedSystem) Receive(message engi.Message) {}

type SpeedComponent struct {
	engi.Point
}

func (*SpeedComponent) Type() string {
	return "SpeedComponent"
}

type BallSystem struct {
	ecs.LinearSystem
}

func (*BallSystem) Type() string { return "BallSystem" }
func (*BallSystem) Pre()         {}
func (*BallSystem) Post()        {}

func (bs *BallSystem) New(*ecs.World) {}

func (bs *BallSystem) UpdateEntity(entity *ecs.Entity, dt float32) {
	var space *engi.SpaceComponent
	var speed *SpeedComponent
	if !entity.Component(&space) || !entity.Component(&speed) {
		return
	}

	if space.Position.X < 0 {
		engi.Mailbox.Dispatch(ScoreMessage{1})

		space.Position.X = 400 - 16
		space.Position.Y = 400 - 16
		speed.X = 800 * rand.Float32()
		speed.Y = 800 * rand.Float32()
	}

	if space.Position.Y < 0 {
		space.Position.Y = 0
		speed.Y *= -1
	}

	if space.Position.X > (800 - 16) {
		engi.Mailbox.Dispatch(ScoreMessage{2})

		space.Position.X = 400 - 16
		space.Position.Y = 400 - 16
		speed.X = 800 * rand.Float32()
		speed.Y = 800 * rand.Float32()
	}

	if space.Position.Y > (800 - 16) {
		space.Position.Y = 800 - 16
		speed.Y *= -1
	}
}

type ControlSystem struct {
	ecs.LinearSystem
}

func (*ControlSystem) Type() string { return "ControlSystem" }
func (*ControlSystem) Pre()         {}
func (*ControlSystem) Post()        {}

func (c *ControlSystem) New(*ecs.World) {}

func (c *ControlSystem) UpdateEntity(entity *ecs.Entity, dt float32) {
	//Check scheme
	// -Move entity based on that
	var control *ControlComponent
	var space *engi.SpaceComponent

	if !entity.Component(&space) || !entity.Component(&control) {
		return
	}
	up := false
	down := false
	if control.Scheme == "WASD" {
		up = engi.Keys.Get(engi.W).Down()
		down = engi.Keys.Get(engi.S).Down()
	} else {
		up = engi.Keys.Get(engi.ArrowUp).Down()
		down = engi.Keys.Get(engi.ArrowDown).Down()
	}

	if up {
		space.Position.Y -= 800 * dt
	}

	if down {
		space.Position.Y += 800 * dt
	}

}

type ControlComponent struct {
	Scheme string
}

func (*ControlComponent) Type() string {
	return "ControlComponent"
}

type ScoreSystem struct {
	ecs.LinearSystem
	PlayerOneScore, PlayerTwoScore int
	upToDate                       bool
	scoreLock                      sync.RWMutex
}

func (*ScoreSystem) Type() string { return "ScoreSystem" }
func (*ScoreSystem) Pre()         {}
func (*ScoreSystem) Post()        {}

func (sc *ScoreSystem) New(*ecs.World) {
	sc.upToDate = true
	engi.Mailbox.Listen("ScoreMessage", func(message engi.Message) {
		scoreMessage, isScore := message.(ScoreMessage)
		if !isScore {
			return
		}

		sc.scoreLock.Lock()
		if scoreMessage.Player != 1 {
			sc.PlayerOneScore += 1
		} else {
			sc.PlayerTwoScore += 1
		}
		log.Println("The score is now", sc.PlayerOneScore, "vs", sc.PlayerTwoScore)
		sc.upToDate = false
		sc.scoreLock.Unlock()
	})
}

func (c *ScoreSystem) UpdateEntity(entity *ecs.Entity, dt float32) {
	var render *engi.RenderComponent
	var space *engi.SpaceComponent

	if !entity.Component(&render) || !entity.Component(&space) {
		return
	}

	if !c.upToDate {
		c.scoreLock.RLock()
		render.Label = fmt.Sprintf("%v vs %v", c.PlayerOneScore, c.PlayerTwoScore)
		c.upToDate = true
		c.scoreLock.RUnlock()

		render.SetDrawable(basicFont.Render(render.Label))
		width := len(render.Label) * 20

		space.Position.X = float32(400 - (width / 2))
	}
}

type ScoreMessage struct {
	Player int
}

func (ScoreMessage) Type() string {
	return "ScoreMessage"
}

func main() {
	opts := engi.RunOptions{
		Title:         "Pong Demo",
		Width:         800,
		Height:        800,
		ScaleOnResize: true,
	}
	engi.Open(opts, &PongGame{})
}
