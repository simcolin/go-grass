package main

import (
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ojrac/opensimplex-go"
)

const WINDOW_WIDTH = 1600
const WINDOW_HEIGHT = 900
const PIXEL_SIZE = 5

const WIND_SCALE = 30.0
const WIND_POWER = 0.5

const STRAND_MIN_LENGTH = 4
const STRAND_MAX_LENGTH = 10

var STRAND_COLOR = rl.Color{0, 255, 0, 255}
var BACKGROUND_COLOR = rl.Color{0, 12, 0, 0}

type Strand struct {
	X         int32
	Y         int32
	colors    []rl.Color
	positions []int32
}

func (s *Strand) Update(tilt float64, wind float64) {
	topStrandColor := LerpColor(rl.Color{0, 0, 0, 0}, STRAND_COLOR, (wind + (1-wind)*WIND_POWER))
	for i := (0); i < len(s.colors); i++ {
		offset := math.Round(tilt * wind * 0.5 * math.Pow(float64(i), 1.5))
		s.positions[i*2] = s.X + int32(offset)
		s.positions[i*2+1] = s.Y - int32(i)
		s.colors[i] = LerpColor(rl.Color{0, 0, 0, 0}, topStrandColor, float64(i)/float64(len(s.colors)))
	}
}

func (s *Strand) Draw() {
	for i, color := range s.colors {
		rl.DrawPixel(s.positions[i*2], s.positions[i*2+1], color)
	}
}

func NewStrand() *Strand {
	length := STRAND_MIN_LENGTH + rand.Int31()%(STRAND_MAX_LENGTH-STRAND_MIN_LENGTH)
	return &Strand{
		X:         rand.Int31() % int32(rl.GetScreenWidth()) / PIXEL_SIZE,
		Y:         rand.Int31() % int32(rl.GetScreenHeight()) / PIXEL_SIZE,
		colors:    make([]rl.Color, length),
		positions: make([]int32, length*2),
	}
}

func LerpColor(start, end rl.Color, amount float64) rl.Color {
	return rl.Color{
		start.R + uint8(float64(end.R-start.R)*amount),
		start.G + uint8(float64(end.G-start.G)*amount),
		start.B + uint8(float64(end.B-start.B)*amount),
		start.A + uint8(float64(end.A-start.A)*amount),
	}
}

func main() {
	rl.InitWindow(WINDOW_WIDTH, WINDOW_HEIGHT, "Pixelised Grass")
	defer rl.CloseWindow()
	rl.SetWindowState(rl.FlagWindowResizable)
	// move to second screen
	// rl.SetWindowPosition(
	// 	rl.GetMonitorWidth(0)+(rl.GetMonitorWidth(1)/2)-(WINDOW_WIDTH/2),
	// 	(rl.GetMonitorHeight(1)/2)-(WINDOW_HEIGHT/2),
	// )

	simplex := opensimplex.NewNormalized(rand.Int63())

	// rl.SetTargetFPS(240)

	windPosition := rl.NewVector3(0, 0, 0)
	windDirection := rl.NewVector3(0, 0, 0)
	windDirectionTarget := rl.NewVector3(2, 2, 0.1)

	strands := make([]*Strand, 5000)
	for i := range strands {
		strands[i] = NewStrand()
	}
	sort.SliceStable(strands, func(i, j int) bool {
		return strands[i].Y < strands[j].Y
	})

	mode := "multithread_classic"

	for !rl.WindowShouldClose() {
		if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			newDir := rl.Vector2Subtract(
				rl.NewVector2(float32(rl.GetScreenWidth()/2), float32(rl.GetScreenHeight()/2)),
				rl.GetMousePosition(),
			)
			newDir = rl.Vector2Scale(rl.Vector2Normalize(newDir), 2)
			windDirectionTarget.X = newDir.X
			windDirectionTarget.Y = newDir.Y
		}

		if rl.IsKeyPressed(rl.KeyQ) {
			mode = "multithread_strands"
		} else if rl.IsKeyPressed(rl.KeyW) {
			mode = "multithread_classic"
		} else if rl.IsKeyPressed(rl.KeyE) {
			mode = "single_thread"
		}

		rl.BeginDrawing()
		rl.ClearBackground(BACKGROUND_COLOR)

		rl.PushMatrix()
		rl.Scalef(PIXEL_SIZE, PIXEL_SIZE, 1)

		if mode == "multithread_strands" { // one thread per strand (5000 strands total)
			var waitGroup sync.WaitGroup
			waitGroup.Add(len(strands))
			for _, strand := range strands {
				go func() {
					defer waitGroup.Done()
					noiseValue := simplex.Eval2(
						float64(strand.X)/WIND_SCALE+float64(windPosition.X),
						float64(strand.Y)/WIND_SCALE+float64(windPosition.Y),
					)
					strand.Update(0.5, noiseValue)
				}()
			}

			waitGroup.Wait()

			for _, strand := range strands {
				strand.Draw()
			}
		} else if mode == "multithread_classic" { // 12 thread sharing strands beetween them
			var waitGroup sync.WaitGroup
			numCpu := runtime.NumCPU()
			waitGroup.Add(numCpu)
			for i := 0; i < numCpu; i++ {
				go func() {
					defer waitGroup.Done()
					for j := i; j < len(strands); j += numCpu {
						strand := strands[j]
						noiseValue := simplex.Eval2(
							float64(strand.X)/WIND_SCALE+float64(windPosition.X),
							float64(strand.Y)/WIND_SCALE+float64(windPosition.Y),
						)
						strand.Update(0.5, noiseValue)
					}
				}()
			}

			waitGroup.Wait()

			for _, strand := range strands {
				strand.Draw()
			}
		} else if mode == "single_thread" { // single thread
			for _, strand := range strands {
				noiseValue := simplex.Eval2(
					float64(strand.X)/WIND_SCALE+float64(windPosition.X),
					float64(strand.Y)/WIND_SCALE+float64(windPosition.Y),
				)
				strand.Update(0.5, noiseValue)
				strand.Draw()
			}
		}

		rl.PopMatrix()

		frameTime := rl.GetFrameTime()

		rl.DrawFPS(10, 10)
		rl.DrawText(mode, 10, 40, 16, rl.White)

		windDirection = rl.Vector3Lerp(windDirection, windDirectionTarget, 0.01)
		windPosition = rl.Vector3Add(windPosition, rl.Vector3Scale(windDirection, frameTime))

		rl.EndDrawing()
	}
}
