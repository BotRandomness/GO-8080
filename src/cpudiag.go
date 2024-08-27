package main

import (
	"fmt"
	"os"
)

func (cpu *cpu) runCpudiag() {
		cpu.loadRom("roms/cpudiag/cpudiag.bin", 0x100)

		for {
			if cpu.pc == 0x0689 {
				//fmt.Println("Error: The test at PC:", fmt.Sprintf("%X", prevPC), "failed")
				fmt.Println("Error")
				os.Exit(3)
			}
			if cpu.pc == 0x069B {
				fmt.Println("Success!")
				os.Exit(3)
			}

			cycles := cpu.executeInstruction()
			_ = cycles
		}
}