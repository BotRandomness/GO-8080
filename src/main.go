package main

import (
	"fmt"
	"os"
	"strconv"
)

//general settings
var scale float32 = 2
var debug bool = false
var fps bool = false

func main() {
	fmt.Println("GO-8080")
	cpu := cpu{}
	cpu.cpuInit()
	fmt.Println("Intel8080 init")
	
	state := 0
	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		if args[i] == "-t" {
			state = 1
		} else if args[i] == "-c" {
			state = 2
		} else if args[i] == "-d" {
			debug = true
		} else if args[i] == "-f" {
			fps = true
		} else if args[i] == "-s" {
			scale64, _ := strconv.ParseFloat(args[i + 1], 32)
			scale = float32(scale64)
		}
	}

	if state == 1 {
		cpu.runTST8080()
	} else if state == 2 {
		cpu.runCpudiag()
	} else {
		cpu.playSpaceInvaders()
	}
}