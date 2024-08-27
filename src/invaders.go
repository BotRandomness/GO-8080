package main

import (
	"github.com/gen2brain/raylib-go/raylib"
	"image/color"
)

const cycleMax = 33000
const firstInterruptCycles = cycleMax / 2
const secondInterruptCycles = cycleMax

func (cpu *cpu) executeInterrupt(interruptNumber uint8) {
	if cpu.interruptEnable == true {
		cpu.memory[cpu.sp - 1] = uint8(cpu.pc >> 8)
		cpu.memory[cpu.sp - 2] = uint8(cpu.pc & 0xFF)
		cpu.sp -= 2

		switch interruptNumber {
		case 1:
			cpu.pc = 0x08
		case 2:
			cpu.pc = 0x10
		}

		cpu.interruptEnable = false
	}
}

func (cpu *cpu) portsIN(port uint8)  {
	switch port {
		case 1:
			//port 1 player 1 input
			var port1Bits uint8 = 0x08 //bit 3 = 1
			if rl.IsKeyPressed(rl.KeyC) {      
				port1Bits |= 0x01 //bit 0 = CREDIT (1 if deposit)
			}
			if rl.IsKeyPressed(rl.KeyX) {       
				port1Bits |= 0x04 //bit 2 = 1P start (1 if pressed)
			}
			if rl.IsKeyDown(rl.KeySpace) {        
				port1Bits |= 0x10 //bit 4 = 1P shot (1 if pressed)
			}
			if rl.IsKeyDown(rl.KeyLeft) {        
				port1Bits |= 0x20 //bit 5 = 1P left (1 if pressed)
			}
			if rl.IsKeyDown(rl.KeyRight) {       
				port1Bits |= 0x40 //bit 6 = 1P right (1 if pressed)
			}
			cpu.regs["a"] = port1Bits
		case 3:
			shiftValue := uint16(cpu.shiftReg2)<<8 | uint16(cpu.shiftReg1)
        	cpu.regs["a"] = uint8((shiftValue >> (8 - cpu.shiftOffset)) & 0xFF)
		default:
			cpu.regs["a"] = 0
	}
}

func (cpu *cpu) portsOUT(port uint8) {
	switch port {
		case 2:
			cpu.shiftOffset = cpu.regs["a"] & 0x07
		case 4:
			cpu.shiftReg2 = cpu.shiftReg1
        	cpu.shiftReg1 = cpu.regs["a"]
		default:
			//cpu.regs["a"] = 0
	}
}

func (cpu *cpu) updateScreenBuffer(pixelData []color.RGBA) {
	vramStart := 0x2400
	screenWidth := 224
	screenHeight := 256

	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			byteIndex := vramStart + (y / 8) + ((x) * 32)
			bitIndex := uint8(y % 8)

			pixelColor := (cpu.memory[byteIndex] >> bitIndex) & 0x01

			colorValue := color.RGBA{0, 0, 0, 255}
			if pixelColor > 0 {
				colorValue = color.RGBA{255, 255, 255, 255}
			}

			pixelData[(screenHeight-y-1)*screenWidth+x] = colorValue
		}
	}
}

func (cpu *cpu) playSpaceInvaders() {
	cpu.interruptEnable = true

	cpu.loadRom("roms/invaders/invaders.rom", 0x0000)

	screenWidth := 224 * scale
	screenHeight := 256 * scale
	rl.InitWindow(int32(screenWidth), int32(screenHeight), "SPACE INVADERS (GO-8080 EMU)")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	textureWidth := 224
	textureHeight := 256
	screenImage := rl.GenImageColor(int(textureWidth), int(textureHeight), rl.Black)
	screenTexture := rl.LoadTextureFromImage(screenImage)
	defer rl.UnloadTexture(screenTexture)
	defer rl.UnloadImage(screenImage)

	//buffer to hold the pixel data
	pixelData := make([]color.RGBA, textureWidth*textureHeight)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)
		totalCycles := 0

		for totalCycles < firstInterruptCycles {
			cycles := cpu.executeInstruction()
			totalCycles += cycles
		}

		cpu.executeInterrupt(1)

		for totalCycles < secondInterruptCycles {
			cycles := cpu.executeInstruction()
			totalCycles += cycles
		}

		cpu.executeInterrupt(2)

		//update the pixel data directly
		cpu.updateScreenBuffer(pixelData)

		//update the texture with the new pixel data
		rl.UpdateTexture(screenTexture, pixelData)

		rl.DrawTextureEx(screenTexture, rl.NewVector2(0, 0), 0, scale, rl.White)

		if debug || fps {
			rl.DrawFPS(0, 0)
		}

		rl.EndDrawing()
	}
}