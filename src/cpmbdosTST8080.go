package main

import "fmt"

func (cpu *cpu) cpmBdos() {
	switch cpu.regs["c"] {
	case 0x02:
		fmt.Printf("%c", cpu.regs["e"])
	case 0x09:
		addr := cpu.get16BitReg("de")
		for {
			ch := cpu.memory[addr]
			if ch == '$' {
				break
			}
			fmt.Printf("%c", ch)
			addr+=1
		}
	}
	//cpu.pc++
}

func (cpu *cpu) runTST8080() {
		cpu.loadRom("roms/TST8080/TST8080.COM", 0x100)
		cpu.pc = 0x0100
		cpu.memory[0x0000] = 0x76 //HLT
		cpu.memory[0x0005] = 0xC9 //RET

		for {
			if cpu.pc == 0x0005 {
				cpu.cpmBdos()
			} else if cpu.pc == 0x0000 {
				break
			}

			cycles := cpu.executeInstruction()
			_ = cycles	
		}
}