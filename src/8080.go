package main

import "fmt"
import "os"
import "io"
import "math/bits"
import "strings"
//import "time"

type cpu struct {
	regs map[string]uint8 //a, b, c, d, e, h, l 8-bit registers
	//a, b, c, d, e, h, l uint8 //8-bit registers
	pc, sp uint16 //special 16-bit registers
	zero, sign, parity, carry, ac bool //flags (Z, S, P, CY, AC)
	memory [65536]uint8 //64KB of memory (0x000-0x1FFF=ROM, 0x2000-0x23FF=RAM, 0x2400-0x3FFF=VRAM, 0x4000-0xFFFF=RAM Mirror)
	
	opcode uint8
	byte2 uint8
	byte3 uint8
	addr uint16
	
	//extra regs/control var for Space Invaders Arcade Cabinet hardware
	interruptEnable bool
	shiftReg1 uint8
	shiftReg2 uint8
	shiftOffset uint8
	controlFlag uint8
}

func (cpu *cpu) cpuInit() {
	cpu.regs = map[string]uint8 {"a":0, "b":0, "c":0, "d":0, "e":0, "h":0, "l":0}
}

func (cpu *cpu) loadRom(romPath string, startAddr int) {
	rom, err := os.Open(romPath)
	if err != nil {
		panic(err)
	}
	defer rom.Close()

	bytes, err := rom.Read(cpu.memory[startAddr:])
	if err != nil && err != io.EOF {
		panic(err)
	}

	fmt.Printf("%v bytes loaded into memory\n", bytes)
}

func (cpu *cpu) dumpMemory(filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	bytes, err := file.Write(cpu.memory[:])
	if err != nil {
		panic(err)
	}

	fmt.Printf("Memory dump to file: %v, wrote %v bytes\n", filePath, bytes)
}

func (cpu *cpu) updateFlagsNOC(value int16) {
	cpu.zero = (value & 0xff) == 0;
    cpu.sign = 0x80 == (value & 0x80);
    cpu.parity = bits.OnesCount16(uint16((value & 0xff))) % 2 == 0; 
    //cpu.carry = value < 0 || value > 0xff;
    cpu.ac = cpu.carry;
}

func (cpu *cpu) updateFlagsNOCAC(value int16) {
	cpu.zero = (value & 0xff) == 0;
    cpu.sign = 0x80 == (value & 0x80);
    cpu.parity = bits.OnesCount16(uint16((value & 0xff))) % 2 == 0; 
    //cpu.carry = value < 0 || value > 0xff;
    //cpu.ac = cpu.carry;
}

func (cpu *cpu) updateFlagsOC(value int16) {
	//cpu.zero = (value & 0xff) == 0;
    //cpu.sign = 0x80 == (value & 0x80);
    //cpu.parity = bits.OnesCount16(uint16((value & 0xff))) % 2 == 0; 
    cpu.carry = value < 0 || value > 0xff;
    //cpu.ac = cpu.carry;
}

func (cpu *cpu) updateFlags(value int16) {
	cpu.zero = (value & 0xff) == 0;
    cpu.sign = 0x80 == (value & 0x80);
    cpu.parity = bits.OnesCount16(uint16((value & 0xff))) % 2 == 0; 
    cpu.carry = value < 0 || value > 0xff;
    cpu.ac = cpu.carry;
}

func (cpu *cpu) get16BitReg(pair string) uint16 {
	switch pair {
		case "bc":
			return uint16(cpu.regs["b"]) << 8 | uint16(cpu.regs["c"])
		case "de":
			return uint16(cpu.regs["d"]) << 8 | uint16(cpu.regs["e"])
		case "hl":
			return uint16(cpu.regs["h"]) << 8 | uint16(cpu.regs["l"])
		default:
			return 0
	}
}

func (cpu *cpu) load16BitReg(pair string, value uint16) {
	switch pair {
		case "bc":
			cpu.regs["b"] = uint8(value >> 8)
			cpu.regs["c"] = uint8(value & 0xFF)
		case "de":
			cpu.regs["d"] = uint8(value >> 8)
			cpu.regs["e"] = uint8(value & 0xFF)
		case "hl":
			cpu.regs["h"] = uint8(value >> 8)
			cpu.regs["l"] = uint8(value & 0xFF)
	}
}

func (cpu *cpu) trace(bytes int, mnemonic string) {
	if debug {
	    var ct, pt, st, zt int
	    if cpu.carry {
	        ct = 1
	    } else {
	        ct = 0
	    }
	    if cpu.parity {
	        pt = 1
	    } else {
	        pt = 0
	    }
	    if cpu.sign {
	        st = 1
	    } else {
	        st = 0
	    }
	    if cpu.zero {
	        zt = 1
	    } else {
	        zt = 0
	    }

	    mnemonic = strings.ReplaceAll(mnemonic, "d8", fmt.Sprintf("%X", cpu.byte2))
	    mnemonic = strings.ReplaceAll(mnemonic, "d16", fmt.Sprintf("%X%X", cpu.byte3, cpu.byte2))
	    mnemonic = strings.ReplaceAll(mnemonic, "addr", fmt.Sprintf("%X%X", cpu.byte3, cpu.byte2))
	    mnemonic = strings.ReplaceAll(mnemonic, "a", fmt.Sprintf("%X", cpu.regs["a"]))
	    mnemonic = strings.ReplaceAll(mnemonic, "b", fmt.Sprintf("%X", cpu.regs["b"]))
	    mnemonic = strings.ReplaceAll(mnemonic, "c", fmt.Sprintf("%X", cpu.regs["c"]))
	    mnemonic = strings.ReplaceAll(mnemonic, "d", fmt.Sprintf("%X", cpu.regs["d"]))
	    mnemonic = strings.ReplaceAll(mnemonic, "e", fmt.Sprintf("%X", cpu.regs["e"]))
	    mnemonic = strings.ReplaceAll(mnemonic, "h", fmt.Sprintf("%X", cpu.regs["h"]))
	    mnemonic = strings.ReplaceAll(mnemonic, "l", fmt.Sprintf("%X", cpu.regs["l"]))
	    mnemonic = strings.ReplaceAll(mnemonic, "pc", fmt.Sprintf("%X", cpu.pc))
	    mnemonic = strings.ReplaceAll(mnemonic, "sp", fmt.Sprintf("%X", cpu.sp))

	    var bc uint16 = cpu.get16BitReg("bc")
	    var de uint16 = cpu.get16BitReg("de")
	    var hl uint16 = cpu.get16BitReg("hl")

	    switch bytes {
	    case 1:
	        fmt.Printf("A:%-2v C:%-2v P:%-2v S:%-2v Z:%-2v BC:%-4v DE:%-4v HL:%-4v SP:%-4v  %-4v %-4v %-4v %-4v %-9v\n",
	            fmt.Sprintf("%X", cpu.regs["a"]),
	            ct, pt, st, zt,
	            fmt.Sprintf("%X", bc),
	            fmt.Sprintf("%X", de),
	            fmt.Sprintf("%X", hl),
	            fmt.Sprintf("%X", cpu.sp),
	            fmt.Sprintf("%X", cpu.pc),
	            fmt.Sprintf("%X", cpu.opcode),
	            "", "", mnemonic)
	    case 2:
	        fmt.Printf("A:%-2v C:%-2v P:%-2v S:%-2v Z:%-2v BC:%-4v DE:%-4v HL:%-4v SP:%-4v  %-4v %-4v %-4v %-4v %-9v\n",
	            fmt.Sprintf("%X", cpu.regs["a"]),
	            ct, pt, st, zt,
	            fmt.Sprintf("%X", bc),
	            fmt.Sprintf("%X", de),
	            fmt.Sprintf("%X", hl),
	            fmt.Sprintf("%X", cpu.sp),
	            fmt.Sprintf("%X", cpu.pc),
	            fmt.Sprintf("%X", cpu.opcode),
	            fmt.Sprintf("%X", cpu.byte2),
	            "", mnemonic)
	    case 3:
	        fmt.Printf("A:%-2v C:%-2v P:%-2v S:%-2v Z:%-2v BC:%-4v DE:%-4v HL:%-4v SP:%-4v  %-4v %-4v %-4v %-4v %-9v\n",
	            fmt.Sprintf("%X", cpu.regs["a"]),
	            ct, pt, st, zt,
	            fmt.Sprintf("%X", bc),
	            fmt.Sprintf("%X", de),
	            fmt.Sprintf("%X", hl),
	            fmt.Sprintf("%X", cpu.sp),
	            fmt.Sprintf("%X", cpu.pc),
	            fmt.Sprintf("%X", cpu.opcode),
	            fmt.Sprintf("%X", cpu.byte2),
	            fmt.Sprintf("%X", cpu.byte3),
	            mnemonic)
	    }
	}
}

func (cpu *cpu) NOP() int {
	cycle := 4
	cpu.pc++
	return cycle
}
func (cpu *cpu) MOVR1R2(r1 string, r2 string) int {
	cycle := 5
	cpu.regs[r1] = cpu.regs[r2]
	cpu.pc++
	return cycle
}
func (cpu *cpu) MOVRM(r string) int {
	cycle := 7
	cpu.regs[r] = cpu.memory[cpu.get16BitReg("hl")]
	cpu.pc++
	return cycle
}
func (cpu *cpu) MOVMR(r string) int {
	cycle := 7
	cpu.memory[cpu.get16BitReg("hl")] = cpu.regs[r]
	cpu.pc++
	return cycle
}
func (cpu *cpu) MVIRD8(r string) int {
	cycle := 7
	cpu.regs[r] = cpu.byte2
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) MVIMD8() int {
	cycle := 10
	cpu.memory[cpu.get16BitReg("hl")] = cpu.byte2
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) LXIRPD16(rh string, rl string) int {
	cycle := 10
	cpu.regs[rh] = cpu.byte3
	cpu.regs[rl] = cpu.byte2
	cpu.pc += 3
	return cycle
}
func (cpu *cpu) LXISPD16() int {
	cycle := 10
	cpu.sp = cpu.addr 	
	cpu.pc += 3
	return cycle
}
func (cpu *cpu) CALL() int {
	cycle := 17
	returnAddr := cpu.pc + 3
	cpu.memory[cpu.sp - 1] = uint8(returnAddr >> 8)
	cpu.memory[cpu.sp - 2] = uint8(returnAddr & 0xFF)
	cpu.sp -= 2
	cpu.pc = cpu.addr
	return cycle
}
func (cpu *cpu) RET() int {
	cycle := 10
	lowByte := uint16(cpu.memory[cpu.sp])
    highByte := uint16(cpu.memory[cpu.sp+1]) << 8
    cpu.pc = lowByte | highByte
    cpu.sp += 2
    return cycle
}
func (cpu *cpu) JMP() int {
	cycle := 10
	cpu.pc = cpu.addr
	return cycle
}
func (cpu *cpu) ANI() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) & int16(cpu.byte2)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	//cpu.carry = false
	//cpu.ac = false
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) JCON(flag bool, condition bool) int {
	cycle := 10
	if flag == condition {
		cpu.pc = cpu.addr
	} else {
		cpu.pc += 3
	}
	return cycle
}
func (cpu *cpu) ADI() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) + int16(cpu.byte2)
	cpu.regs["a"] = uint8(result & 0xFF)
	cpu.updateFlags(result)
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) CPI() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) - int16(cpu.byte2)
	cpu.updateFlags(result)
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) ACI() int {
	cycle := 7
	var cv uint8
	if cpu.carry {
		cv = 1
	} else {
		cv = 0
	}
	var result int16 = int16(cpu.regs["a"]) + int16(cpu.byte2) + int16(cv)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) SUI() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) - int16(cpu.byte2)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) SBI() int {
	cycle := 7
	var cv uint8
	if cpu.carry {
		cv = 1
	} else {
		cv = 0
	}
	var result int16 = int16(cpu.regs["a"]) - int16(cpu.byte2) - int16(cv)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) ORI() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) | int16(cpu.byte2)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) XRI() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) ^ int16(cpu.byte2)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) CCON(flag bool) int {
	cycle := 17
	if flag {
		returnAddr := cpu.pc + 3
		cpu.memory[cpu.sp - 1] = uint8(returnAddr >> 8)
		cpu.memory[cpu.sp - 2] = uint8(returnAddr & 0xFF)
		cpu.sp -= 2
		cpu.pc = cpu.addr
	} else {
		cycle = 11
		cpu.pc += 3
	}

	return cycle
}
func (cpu *cpu) RCON(flag bool, condition bool) int {
	cycle := 11
	if flag == condition {
		lowByte := uint16(cpu.memory[cpu.sp])
        highByte := uint16(cpu.memory[cpu.sp+1]) << 8
        cpu.pc = lowByte | highByte
        cpu.sp += 2
	} else {
		cycle = 5
		cpu.pc += 1
	}
	return cycle
}
func (cpu *cpu) INRR(r string) int {
	cycle := 5
	var result int16 = int16(cpu.regs[r]) + int16(1)
	cpu.regs[r] = uint8(result & 0xFF)
	cpu.updateFlagsNOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) DCRR(r string) int {
	cycle := 5
	var result int16 = int16(cpu.regs[r]) - int16(1)
	cpu.regs[r] = uint8(result & 0xFF)
	cpu.updateFlagsNOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) XRAR(r string) int {
	cycle := 4
	var result int16 = int16(cpu.regs["a"]) ^ int16(cpu.regs[r])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	//cpu.carry = false
	//cpu.ac = false
	cpu.pc++
	return cycle
}
func (cpu *cpu) ADDR(r string) int {
	cycle := 4
	var result int16 = int16(cpu.regs["a"]) + int16(cpu.regs[r])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) SUBR(r string) int {
	cycle := 4
	var result int16 = int16(cpu.regs["a"]) - int16(cpu.regs[r])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) ADCR(r string) int {
	cycle := 4
	var cv uint8
	if cpu.carry {
		cv = 1
	} else {
		cv = 0
	}
	var result int16 = int16(cpu.regs["a"]) + int16(cpu.regs[r]) + int16(cv)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) SBBR(r string) int {
	cycle := 4
	var cv uint8
	if cpu.carry {
		cv = 1
	} else {
		cv = 0
	}
	var result int16 = int16(cpu.regs["a"]) - int16(cpu.regs[r]) - int16(cv)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) ANAR(r string) int {
	cycle := 4
	var result int16 = int16(cpu.regs["a"]) & int16(cpu.regs[r])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) ORAR(r string) int {
	cycle := 4
	var result int16 = int16(cpu.regs["a"]) | int16(cpu.regs[r])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) CMPR(r string) int {
	cycle := 4
	var result int16 = int16(cpu.regs["a"]) - int16(cpu.regs[r])
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) CMPM() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) - int16(cpu.memory[cpu.get16BitReg("hl")])
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) ADDM() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) + int16(cpu.memory[cpu.get16BitReg("hl")])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) SUBM() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) - int16(cpu.memory[cpu.get16BitReg("hl")])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) ADCM() int {
	cycle := 7
	var cv uint8
	if cpu.carry {
		cv = 1
	} else {
		cv = 0
	}
	var result int16 = int16(cpu.regs["a"]) + int16(cpu.memory[cpu.get16BitReg("hl")]) + int16(cv)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) SBBM() int {
	cycle := 7
	var cv uint8
	if cpu.carry {
		cv = 1
	} else {
		cv = 0
	}
	var result int16 = int16(cpu.regs["a"]) - int16(cpu.memory[cpu.get16BitReg("hl")]) - int16(cv)
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) ANAM() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) & int16(cpu.memory[cpu.get16BitReg("hl")])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) ORAM() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) | int16(cpu.memory[cpu.get16BitReg("hl")])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) XRAM() int {
	cycle := 7
	var result int16 = int16(cpu.regs["a"]) ^ int16(cpu.memory[cpu.get16BitReg("hl")])
	cpu.regs["a"] = uint8(result)
	cpu.updateFlags(result)
	//cpu.carry = false
	//cpu.ac = false
	cpu.pc++
	return cycle
}
func (cpu *cpu) INRM() int {
	cycle := 10
	var result int16 = int16(cpu.memory[cpu.get16BitReg("hl")]) + int16(1)
	cpu.memory[cpu.get16BitReg("hl")] = uint8(result & 0xFF)
	cpu.updateFlagsNOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) DCRM() int {
	cycle := 10
	var result int16 = int16(cpu.memory[cpu.get16BitReg("hl")]) - int16(1)
	cpu.memory[cpu.get16BitReg("hl")] = uint8(result & 0xFF)
	cpu.updateFlagsNOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) INXRP(rp string) int {
	cycle := 5
	var result int16 = int16(cpu.get16BitReg(rp)) + int16(1)
	cpu.load16BitReg(rp, uint16(result))
	//cpu.updateFlagsNOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) DCXRP(rp string) int {
	cycle := 5
	var result int16 = int16(cpu.get16BitReg(rp)) - int16(1)
	cpu.load16BitReg(rp, uint16(result))
	//cpu.updateFlagsNOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) STAADDR() int {
	cycle := 13
	cpu.memory[cpu.addr] = cpu.regs["a"]
	cpu.pc += 3
	return cycle
}
func (cpu *cpu) LDAADDR() int {
	cycle := 13
	cpu.regs["a"] = cpu.memory[cpu.addr]
	cpu.pc += 3
	return cycle
}
func (cpu *cpu) LHLDADDR() int {
	cycle := 16
	cpu.regs["l"] = cpu.memory[cpu.addr]
	cpu.regs["h"] = cpu.memory[cpu.addr + 1]
	cpu.pc += 3
	return cycle
}
func (cpu *cpu) SHLDADDR() int {
	cycle := 16
	cpu.memory[cpu.addr] = cpu.regs["l"]
	cpu.memory[cpu.addr + 1] = cpu.regs["h"]
	cpu.pc += 3
	return cycle
}
func (cpu *cpu) LDAXRP(rp string) int {
	cycle := 7
	cpu.regs["a"] = cpu.memory[cpu.get16BitReg(rp)]
	cpu.pc++
	return cycle
}
func (cpu *cpu) STAXRP(rp string) int {
	cycle := 7
	cpu.memory[cpu.get16BitReg(rp)] = cpu.regs["a"]
	cpu.pc++
	return cycle
}
func (cpu *cpu) XCHG() int {
	cycle := 5
	tempH := cpu.regs["h"]
	tempL := cpu.regs["l"]
	cpu.regs["h"] = cpu.regs["d"]
	cpu.regs["l"] = cpu.regs["e"]
	cpu.regs["d"] = tempH
	cpu.regs["e"] = tempL
	cpu.pc++
	return cycle
}
func (cpu *cpu) DADRP(rp string) int {
	cycle := 18
	var result int16 = int16(cpu.get16BitReg("hl")) + int16(cpu.get16BitReg(rp))
	cpu.load16BitReg("hl", uint16(result))
	cpu.updateFlagsOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) STC() int {
	cycle := 4
	cpu.carry = true
	cpu.pc++
	return cycle
}
func (cpu *cpu) CMC() int {
	cycle := 4
	cpu.carry = !cpu.carry
	cpu.pc++
	return cycle
}
func (cpu *cpu) CMA() int {
	cycle := 4
	cpu.regs["a"] = ^cpu.regs["a"]
	cpu.pc++
	return cycle
}
func (cpu *cpu) DAA() int {
	cycle := 4
	accumulatorValue := cpu.regs["a"]

	if (accumulatorValue & 0x0F) > 9 || cpu.ac {
		accumulatorValue += 0x06
	}
	if (accumulatorValue & 0xF0) > 0x90 || cpu.carry {
		accumulatorValue += 0x60
		cpu.carry = true // Set carry if adjustment causes it
	}

	cpu.regs["a"] = accumulatorValue
	cpu.ac = (accumulatorValue & 0x0F) < 0x06
	cpu.updateFlagsNOCAC(int16(accumulatorValue))
	cpu.pc++
	return cycle
}
func (cpu *cpu) RLC() int {
	cycle := 4
    accumulatorValue := cpu.regs["a"]
    highOrderBit := (accumulatorValue & 0x80) >> 7
    rotatedValue := (accumulatorValue << 1) | highOrderBit
    cpu.carry = (highOrderBit == 1)
    cpu.regs["a"] = rotatedValue
    cpu.pc++
    return cycle
}
func (cpu *cpu) RRC() int {
	cycle := 4
    accumulatorValue := cpu.regs["a"]
    lowOrderBit := accumulatorValue & 0x01
    rotatedValue := (accumulatorValue >> 1) | (lowOrderBit << 7)
    cpu.carry = (lowOrderBit == 1)
    cpu.regs["a"] = rotatedValue
    cpu.pc++
    return cycle
}
func (cpu *cpu) RAL() int {
	cycle := 4
	var cy uint8
	if cpu.carry {
		cy = 1
	} else {
		cy = 0
	}

    accumulatorValue := cpu.regs["a"]
    rotatedValue := (accumulatorValue << 1) | cy
    cpu.carry = (accumulatorValue & 0x80) != 0
    cpu.regs["a"] = rotatedValue
    cpu.pc++
    return cycle
}
func (cpu *cpu) RAR() int {
	cycle := 4
	var cy uint8
	if cpu.carry {
		cy = 1
	} else {
		cy = 0
	}
    accumulatorValue := cpu.regs["a"]
    lowOrderBit := accumulatorValue & 0x01
    rotatedValue := (accumulatorValue >> 1) | (cy << 7)
    cpu.carry = (lowOrderBit == 1)
    cpu.regs["a"] = rotatedValue
    cpu.pc ++
    return cycle
}
func (cpu *cpu) PUSHRP(rh string, rl string) int {
	cycle := 11
	cpu.memory[cpu.sp - 1] = cpu.regs[rh]
	cpu.memory[cpu.sp - 2] = cpu.regs[rl]
	cpu.sp -= 2
	cpu.pc++
	return cycle 
}
func (cpu *cpu) PUSHPSW() int {
	cycle := 11
	var flag uint8 = 0
	if cpu.zero {
		flag = flag | 0x40 //bit 6
	}
	if cpu.sign {
		flag = flag | 0x80 //bit 7
	}
	if cpu.parity {
		flag = flag | 0x04 //bit 2
	}
	if cpu.carry {
		flag = flag | 0x01 //bit 0
	}
	if cpu.ac {
		flag = flag | 0x10 //bit 4
	}
	flag = flag | 0x02 //bit 1 (always 1)
	cpu.memory[cpu.sp - 2] = flag
	cpu.memory[cpu.sp - 1] = cpu.regs["a"]
	cpu.sp -= 2
	cycle = 11
	cpu.pc++
	return cycle
}
func (cpu *cpu) POPPSW() int {
	cycle := 10
	flagByte := cpu.memory[cpu.sp]
	cpu.carry = (flagByte & 0x01) != 0    //bit 0
	cpu.ac = (flagByte & 0x10) != 0       //bit 4
	cpu.parity = (flagByte & 0x04) != 0   //bit 2
	cpu.zero = (flagByte & 0x40) != 0     //bit 6
	cpu.sign = (flagByte & 0x80) != 0     //bit 7
	cpu.regs["a"] = cpu.memory[cpu.sp + 1]
	cpu.sp += 2
	cycle = 10
	cpu.pc++
	return cycle
}
func (cpu *cpu) POPRP(rh string, rl string) int {
	cycle := 10
	cpu.regs[rl] = cpu.memory[cpu.sp]
	cpu.regs[rh] = cpu.memory[cpu.sp + 1]
	cpu.sp += 2
	cpu.pc++
	return cycle
}
func (cpu *cpu) DADSP() int {
	cycle := 10
	var result int16 = int16(cpu.get16BitReg("hl")) + int16(cpu.sp)
	cpu.load16BitReg("hl", uint16(result))
	cpu.updateFlagsOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) DCXSP() int {
	cycle := 5
	var result int16 = int16(cpu.sp) - int16(1)
	cpu.sp = uint16(result)
	//cpu.updateFlagsNOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) INXSP() int {
	cycle := 5
	var result int16 = int16(cpu.sp) + int16(1)
	cpu.sp = uint16(result)
	//cpu.updateFlagsNOC(result)
	cpu.pc++
	return cycle
}
func (cpu *cpu) SPHL() int {
	cycle := 5
	cpu.sp = cpu.get16BitReg("hl")
	cpu.pc++
	return cycle
}
func (cpu *cpu) XTHL() int {
	cycle := 18
	tempL := cpu.regs["l"]
	tempH := cpu.regs["h"]
	cpu.regs["l"] = cpu.memory[cpu.sp]
	cpu.regs["h"] = cpu.memory[cpu.sp + 1]
	cpu.memory[cpu.sp] = tempL
	cpu.memory[cpu.sp + 1] = tempH
	cpu.pc++
	return cycle
}
func (cpu *cpu) PCHL() int {
	cycle := 5
	cpu.pc = cpu.get16BitReg("hl")
	return cycle
}
func (cpu *cpu) EI() int {
	cycle := 4
	cpu.interruptEnable = true
	cpu.pc++
	return cycle
}
func (cpu *cpu) IN() int {
	cycle := 10
	port := cpu.byte2
	cpu.portsIN(port)
	cpu.pc += 2
	return cycle
}
func (cpu *cpu) OUT() int {
	cycle := 10
	port := cpu.byte2
	cpu.portsOUT(port)
	cpu.pc += 2
	return cycle
}

func (cpu *cpu) executeInstruction() int {
	cpu.opcode = cpu.memory[cpu.pc]
	cpu.byte2 = cpu.memory[cpu.pc + 1]
	cpu.byte3 = cpu.memory[cpu.pc + 2]
	cpu.addr = uint16(cpu.memory[cpu.pc + 1]) | (uint16(cpu.memory[cpu.pc + 2]) << 8)
	cycle := 0

	//prevPC := cpu.pc

	switch cpu.opcode {
		case 0x00:
			cpu.trace(1, "NOP")
			cycle = cpu.NOP()
		case 0xC3:
			cpu.trace(3, "JMP addr")
			cycle = cpu.JMP()
		case 0x31:
			cpu.trace(3, "LXI (SP)sp, d16")
			cycle = cpu.LXISPD16()
		case 0xCD:
			cpu.trace(3, "CALL addr")
			cycle = cpu.CALL()
		case 0xE6:
			cpu.trace(2, "ANI d8")
			cycle = cpu.ANI()
		case 0xCA:
			cpu.trace(3, "JZ addr")
			cycle = cpu.JCON(cpu.zero, true)
		case 0xD2:
			cpu.trace(3, "JNC addr")
			cycle = cpu.JCON(cpu.carry, false)
		case 0xEA:
			cpu.trace(3, "JPE addr")
			cycle = cpu.JCON(cpu.parity, true)
		case 0xE2:
			cpu.trace(3, "JPO addr")
			cycle = cpu.JCON(cpu.parity, false)
		case 0xDA:
			cpu.trace(3, "JC addr")
			cycle = cpu.JCON(cpu.carry, true)
		case 0xF2:
			cpu.trace(3, "JP addr")
			cycle = cpu.JCON(cpu.sign, false)
		case 0xC2:
			cpu.trace(3, "JNZ addr")
			cycle = cpu.JCON(cpu.zero, false)
		case 0xFA:
			cpu.trace(3, "JM addr")
			cycle = cpu.JCON(cpu.sign, true)
		case 0xC6:
			cpu.trace(2, "ADI d8")
			cycle = cpu.ADI()
		case 0xFE:
			cpu.trace(2, "CPI d8")
			cycle = cpu.CPI()
		case 0xCE:
			cpu.trace(2, "ACI d8")
			cycle = cpu.ACI()
		case 0xD6:
			cpu.trace(2, "SUI d8")
			cycle = cpu.SUI()
		case 0xDE:
			cpu.trace(2, "SBI d8")
			cycle = cpu.SBI()
		case 0xF6:
			cpu.trace(2, "ORI d8")
			cycle = cpu.ORI()
		case 0xEE:
			cpu.trace(2, "XRI d8")
			cycle = cpu.XRI()
		case 0xDC:
			cpu.trace(3, "CC addr")
			cycle = cpu.CCON(cpu.carry)
		case 0xE4:
			cpu.trace(3, "CPO addr")
			cycle = cpu.CCON(!cpu.parity)
		case 0xFC:
			cpu.trace(3, "CM addr")
			cycle = cpu.CCON(cpu.sign)
		case 0xC4:
			cpu.trace(3, "CNZ addr")
			cycle = cpu.CCON(!cpu.zero)
		case 0xD4:
			cpu.trace(3, "CNC addr")
			cycle = cpu.CCON(!cpu.carry)
		case 0xEC:
			cpu.trace(3, "CPE addr")
			cycle = cpu.CCON(cpu.parity)
		case 0xF4:
			cpu.trace(3, "CP addr")
			cycle = cpu.CCON(!cpu.sign)
		case 0xCC:
			cpu.trace(3, "CZ addr")
			cycle = cpu.CCON(cpu.zero)
		case 0xE8:
			cpu.trace(1, "RPE")
			cycle = cpu.RCON(cpu.parity, true)
		case 0xC8:
			cpu.trace(1, "RZ")
			cycle = cpu.RCON(cpu.zero, true)
		case 0xE0:
			cpu.trace(1, "RPO")
			cycle = cpu.RCON(cpu.parity, false)
		case 0xF0:
			cpu.trace(1, "RP")
			cycle = cpu.RCON(cpu.sign, false)
		case 0xF8:
			cpu.trace(1, "RM")
			cycle = cpu.RCON(cpu.sign, true)
		case 0xD8:
			cpu.trace(1, "RC")
			cycle = cpu.RCON(cpu.carry, true)
		case 0xD0:
			cpu.trace(1, "RNC")
			cycle = cpu.RCON(cpu.carry, false)
		case 0xC0:
			cpu.trace(1, "RNZ")
			cycle = cpu.RCON(cpu.zero, false)
		case 0x3E:
			cpu.trace(2, "MVI (A)a, d8")
			cycle = cpu.MVIRD8("a")
		case 0x3C:
			cpu.trace(1, "INR (A)a")
			cycle = cpu.INRR("a")
		case 0x47:
			cpu.trace(1, "MOV (B)b, (A)a")
			cycle = cpu.MOVR1R2("b", "a")
		case 0x04:
			cpu.trace(1, "INR (B)b")
			cycle = cpu.INRR("b")
		case 0x48:
			cpu.trace(1, "MOV (C)c, (B)b")
			cycle = cpu.MOVR1R2("c", "b")
		case 0x0D:
			cpu.trace(1, "DCR (C)c")
			cycle = cpu.DCRR("c")
		case 0x51:
			cpu.trace(1, "MOV (D)d, (C)c")
			cycle = cpu.MOVR1R2("d", "c")
		case 0x5A:
			cpu.trace(1, "MOV (E)e, (D)d")
			cycle = cpu.MOVR1R2("e", "d")
		case 0x63:
			cpu.trace(1, "MOV (H)h, (E)e")
			cycle = cpu.MOVR1R2("h", "e")
		case 0x6C:
			cpu.trace(1, "MOV (L)l, (H)h")
			cycle = cpu.MOVR1R2("l", "h")
		case 0x7D:
			cpu.trace(1, "MOV (A)a, (L)l")
			cycle = cpu.MOVR1R2("a", "l")
		case 0x3D:
			cpu.trace(1, "DCR (A)a")
			cycle = cpu.DCRR("a")
		case 0x4F:
			cpu.trace(1, "MOV (C)c, (A)a")
			cycle = cpu.MOVR1R2("c", "a")
		case 0x59:
			cpu.trace(1, "MOV (E)e, (C)c")
			cycle = cpu.MOVR1R2("e", "c")
		case 0x6B:
			cpu.trace(1, "MOV (L)l, (E)e")
			cycle = cpu.MOVR1R2("l", "e")
		case 0x45:
			cpu.trace(1, "MOV (B)b, (L)l")
			cycle = cpu.MOVR1R2("b", "l")
		case 0x50:
			cpu.trace(1, "MOV (D)d, (B)b")
			cycle = cpu.MOVR1R2("d", "b")
		case 0x62:
			cpu.trace(1, "MOV (H)h, (D)d")
			cycle = cpu.MOVR1R2("h", "d")
		case 0x7C:
			cpu.trace(1, "MOV (A)a, (H)h")
			cycle = cpu.MOVR1R2("a", "h")
		case 0x57:
			cpu.trace(1, "MOV (D)d, (A)a")
			cycle = cpu.MOVR1R2("d", "a")
		case 0x14:
			cpu.trace(1, "INR (D)d")
			cycle = cpu.INRR("d")
		case 0x6A:
			cpu.trace(1, "MOV (L)l, (D)d")
			cycle = cpu.MOVR1R2("l", "d")
		case 0x4D:
			cpu.trace(1, "MOV (C)c, (L)l")
			cycle = cpu.MOVR1R2("c", "l")
		case 0x0C:
			cpu.trace(1, "INR (C)c")
			cycle = cpu.INRR("c")
		case 0x61:
			cpu.trace(1, "MOV (H)h, (C)c")
			cycle = cpu.MOVR1R2("h", "c")
		case 0x44:
			cpu.trace(1, "MOV (B)b, (H)h")
			cycle = cpu.MOVR1R2("b", "h")
		case 0x05:
			cpu.trace(1, "DCR (B)b")
			cycle = cpu.DCRR("b")
		case 0x58:
			cpu.trace(1, "MOV (E)e, (B)b")
			cycle = cpu.MOVR1R2("e", "b")
		case 0x7B:
			cpu.trace(1, "MOV (A)a, (E)e")
			cycle = cpu.MOVR1R2("a", "e")
		case 0x5F:
			cpu.trace(1, "MOV (E)e, (A)a")
			cycle = cpu.MOVR1R2("e", "a")
		case 0x1C:
			cpu.trace(1, "INR (E)e")
			cycle = cpu.INRR("e")
		case 0x43:
			cpu.trace(1, "MOV (B)b, (E)e")
			cycle = cpu.MOVR1R2("b", "e")
		case 0x60:
			cpu.trace(1, "MOV (H)h, (B)b")
			cycle = cpu.MOVR1R2("h", "b")
		case 0x24:
			cpu.trace(1, "INR (H)h")
			cycle = cpu.INRR("h")
		case 0x4C:
			cpu.trace(1, "MOV (C)c, (H)h")
			cycle = cpu.MOVR1R2("c", "h")
		case 0x69:
			cpu.trace(1, "MOV (L)l, (C)c")
			cycle = cpu.MOVR1R2("l", "c")
		case 0x55:
			cpu.trace(1, "MOV (D)d, (L)l")
			cycle = cpu.MOVR1R2("d", "l")
		case 0x15:
			cpu.trace(1, "DCR (D)d")
			cycle = cpu.DCRR("d")
		case 0x7A:
			cpu.trace(1, "MOV (A)a, (D)d")
			cycle = cpu.MOVR1R2("a", "d")
		case 0x67:
			cpu.trace(1, "MOV (H)h, (A)a")
			cycle = cpu.MOVR1R2("h", "a")
		case 0x25:
			cpu.trace(1, "DCR (H)h")
			cycle = cpu.DCRR("h")
		case 0x54:
			cpu.trace(1, "MOV (D)d, (H)h")
			cycle = cpu.MOVR1R2("d", "h")
		case 0x42:
			cpu.trace(1, "MOV (B)b, (D)d")
			cycle = cpu.MOVR1R2("b", "d")
		case 0x68:
			cpu.trace(1, "MOV (L)l, (B)b")
			cycle = cpu.MOVR1R2("l", "b")
		case 0x2C:
			cpu.trace(1, "INR (L)l")
			cycle = cpu.INRR("l")
		case 0x5D:
			cpu.trace(1, "MOV (E)e, (L)l")
			cycle = cpu.MOVR1R2("e", "l")
		case 0x1D:
			cpu.trace(1, "DCR (E)e")
			cycle = cpu.DCRR("e")
		case 0x4B:
			cpu.trace(1, "MOV (C)c, (E)e")
			cycle = cpu.MOVR1R2("c", "e")
		case 0x79:
			cpu.trace(1, "MOV (A)a, (C)c")
			cycle = cpu.MOVR1R2("a", "c")
		case 0x6F:
			cpu.trace(1, "MOV (L)l, (A)a")
			cycle = cpu.MOVR1R2("l", "a")
		case 0x2D:
			cpu.trace(1, "DCR (L)l")
			cycle = cpu.DCRR("l")
		case 0x65:
			cpu.trace(1, "MOV (H)h, (L)l")
			cycle = cpu.MOVR1R2("h", "l")
		case 0x5C:
			cpu.trace(1, "MOV (E)e, (H)h")
			cycle = cpu.MOVR1R2("e", "h")
		case 0x53:
			cpu.trace(1, "MOV (D)d, (E)e")
			cycle = cpu.MOVR1R2("d", "e")
		case 0x4A:
			cpu.trace(1, "MOV (C)c, (D)d")
			cycle = cpu.MOVR1R2("c", "d")
		case 0x41:
			cpu.trace(1, "MOV (B)b, (C)c")
			cycle = cpu.MOVR1R2("b", "c")
		case 0x78:
			cpu.trace(1, "MOV (A)a, (B)b")
			cycle = cpu.MOVR1R2("a", "b")
		case 0xAF:
			cpu.trace(1, "XRA (A)a")
			cycle = cpu.XRAR("a")
		case 0x06:
			cpu.trace(1, "MVI (B)b, d8")
			cycle = cpu.MVIRD8("b")
		case 0x0E:
			cpu.trace(1, "MVI (C)c, d8")
			cycle = cpu.MVIRD8("c")
		case 0x16:
			cpu.trace(1, "MVI (D)d, d8")
			cycle = cpu.MVIRD8("d")
		case 0x1E:
			cpu.trace(1, "MVI (E)e, d8")
			cycle = cpu.MVIRD8("e")
		case 0x26:
			cpu.trace(1, "MVI (H)h, d8")
			cycle = cpu.MVIRD8("h")
		case 0x2E:
			cpu.trace(1, "MVI (L)l, d8")
			cycle = cpu.MVIRD8("l")
		case 0x80:
			cpu.trace(1, "ADD (B)b")
			cycle = cpu.ADDR("b")
		case 0x81:
			cpu.trace(1, "ADD (C)c")
			cycle = cpu.ADDR("c")
		case 0x82:
			cpu.trace(1, "ADD (D)d")
			cycle = cpu.ADDR("d")
		case 0x83:
			cpu.trace(1, "ADD (E)e")
			cycle = cpu.ADDR("e")
		case 0x84:
			cpu.trace(1, "ADD (H)h")
			cycle = cpu.ADDR("h")
		case 0x85:
			cpu.trace(1, "ADD (L)l")
			cycle = cpu.ADDR("l")
		case 0x87:
			cpu.trace(1, "ADD (A)a")
			cycle = cpu.ADDR("a")
		case 0x90:
			cpu.trace(1, "SUB (B)b")
			cycle = cpu.SUBR("b")
		case 0x91:
			cpu.trace(1, "SUB (C)c")
			cycle = cpu.SUBR("c")
		case 0x92:
			cpu.trace(1, "SUB (D)d")
			cycle = cpu.SUBR("d")
		case 0x93:
			cpu.trace(1, "SUB (E)e")
			cycle = cpu.SUBR("e")
		case 0x94:
			cpu.trace(1, "SUB (H)h")
			cycle = cpu.SUBR("h")
		case 0x95:
			cpu.trace(1, "SUB (L)l")
			cycle = cpu.SUBR("l")
		case 0x97:
			cpu.trace(1, "SUB (A)a")
			cycle = cpu.SUBR("a")
		case 0x88:
			cpu.trace(1, "ADC (B)b")
			cycle = cpu.ADCR("b")
		case 0x89:
			cpu.trace(1, "ADC (C)c")
			cycle = cpu.ADCR("c")
		case 0x8A:
			cpu.trace(1, "ADC (D)d")
			cycle = cpu.ADCR("d")
		case 0x8B:
			cpu.trace(1, "ADC (E)e")
			cycle = cpu.ADCR("e")
		case 0x8C:
			cpu.trace(1, "ADC (H)h")
			cycle = cpu.ADCR("h")
		case 0x8D:
			cpu.trace(1, "ADC (L)l")
			cycle = cpu.ADCR("l")
		case 0x8F:
			cpu.trace(1, "ADC (A)a")
			cycle = cpu.ADCR("a")
		case 0x98:
			cpu.trace(1, "SBB (B)b")
			cycle = cpu.SBBR("b")
		case 0x99:
			cpu.trace(1, "SBB (C)c")
			cycle = cpu.SBBR("c")
		case 0x9A:
			cpu.trace(1, "SBB (D)d")
			cycle = cpu.SBBR("d")
		case 0x9B:
			cpu.trace(1, "SBB (E)e")
			cycle = cpu.SBBR("e")
		case 0x9C:
			cpu.trace(1, "SBB (H)h")
			cycle = cpu.SBBR("h")
		case 0x9D:
			cpu.trace(1, "SBB (L)l")
			cycle = cpu.SBBR("l")
		case 0x9F:
			cpu.trace(1, "SBB (A)a")
			cycle = cpu.SBBR("a")
		case 0xA7:
			cpu.trace(1, "ANA (A)a")
			cycle = cpu.ANAR("a")
		case 0xA1:
			cpu.trace(1, "ANA (C)c")
			cycle = cpu.ANAR("c")
		case 0xA2:
			cpu.trace(1, "ANA (D)d")
			cycle = cpu.ANAR("d")
		case 0xA3:
			cpu.trace(1, "ANA (E)e")
			cycle = cpu.ANAR("e")
		case 0xA4:
			cpu.trace(1, "ANA (H)h")
			cycle = cpu.ANAR("h")
		case 0xA5:
			cpu.trace(1, "ANA (L)l")
			cycle = cpu.ANAR("l")
		case 0xB0:
			cpu.trace(1, "ORA (B)b")
			cycle = cpu.ORAR("b")
		case 0xB1:
			cpu.trace(1, "ORA (C)c")
			cycle = cpu.ORAR("c")
		case 0xB2:
			cpu.trace(1, "ORA (D)d")
			cycle = cpu.ORAR("d")
		case 0xB3:
			cpu.trace(1, "ORA (E)e")
			cycle = cpu.ORAR("e")
		case 0xB4:
			cpu.trace(1, "ORA (H)h")
			cycle = cpu.ORAR("h")
		case 0xB5:
			cpu.trace(1, "ORA (L)l")
			cycle = cpu.ORAR("l")
		case 0xB7:
			cpu.trace(1, "ORA (A)a")
			cycle = cpu.ORAR("a")
		case 0xA8:
			cpu.trace(1, "XRA (B)b")
			cycle = cpu.XRAR("b")
		case 0xA9:
			cpu.trace(1, "XRA (C)c")
			cycle = cpu.XRAR("c")
		case 0xAA:
			cpu.trace(1, "XRA (D)d")
			cycle = cpu.XRAR("d")
		case 0xAB:
			cpu.trace(1, "XRA (E)e")
			cycle = cpu.XRAR("e")
		case 0xAC:
			cpu.trace(1, "XRA (H)h")
			cycle = cpu.XRAR("h")
		case 0xAD:
			cpu.trace(1, "XRA (L)l")
			cycle = cpu.XRAR("l")
		case 0x70:
			cpu.trace(1, "MOV M, (B)b")
			cycle = cpu.MOVMR("b")
		case 0x46:
			cpu.trace(1, "MOV (B)b, M")
			cycle = cpu.MOVRM("b")
		case 0xB8:
			cpu.trace(1, "CMP (B)b")
			cycle = cpu.CMPR("b")
		case 0x72:
			cpu.trace(1, "MOV M, (D)d")
			cycle = cpu.MOVMR("d")
		case 0x56:
			cpu.trace(1, "MOV (D)d, M")
			cycle = cpu.MOVRM("d")
		case 0xBA:
			cpu.trace(1, "CMP (D)d")
			cycle = cpu.CMPR("d")
		case 0x73:
			cpu.trace(1, "MOV M, (E)e")
			cycle = cpu.MOVMR("e")
		case 0x5E:
			cpu.trace(1, "MOV (E)e, M")
			cycle = cpu.MOVRM("e")
		case 0xBB:
			cpu.trace(1, "CMP (E)e")
			cycle = cpu.CMPR("e")
		case 0x74:
			cpu.trace(1, "MOV M, (H)h")
			cycle = cpu.MOVMR("h")
		case 0x66:
			cpu.trace(1, "MOV (H)h, M")
			cycle = cpu.MOVRM("h")
		case 0xBC:
			cpu.trace(1, "CMP (H)h")
			cycle = cpu.CMPR("h")
		case 0x75:
			cpu.trace(1, "MOV M, (L)l")
			cycle = cpu.MOVMR("l")
		case 0x6E:
			cpu.trace(1, "MOV (L)l, M")
			cycle = cpu.MOVRM("l")
		case 0xBD:
			cpu.trace(1, "CMP (L)l")
			cycle = cpu.CMPR("l")
		case 0x77:
			cpu.trace(1, "MOV M, (A)a")
			cycle = cpu.MOVMR("a")
		case 0xBE:
			cpu.trace(1, "CMP M")
			cycle = cpu.CMPM()
		case 0x86:
			cpu.trace(1, "ADD M")
			cycle = cpu.ADDM()
		case 0x7E:
			cpu.trace(1, "MOV (A)a, M")
			cycle = cpu.MOVRM("a")
		case 0x96:
			cpu.trace(1, "SUB M")
			cycle = cpu.SUBM()
		case 0x8E:
			cpu.trace(1, "ADC M")
			cycle = cpu.ADCM()
		case 0x9E:
			cpu.trace(1, "SBB M")
			cycle = cpu.SBBM()
		case 0xA6:
			cpu.trace(1, "ANA M")
			cycle = cpu.ANAM()
		case 0xB6:
			cpu.trace(1, "ORA M")
			cycle = cpu.ORAM()
		case 0xAE:
			cpu.trace(1, "XRA M")
			cycle = cpu.XRAM()
		case 0x36:
			cpu.trace(2, "MVI M, d8")
			cycle = cpu.MVIMD8()
		case 0x34:
			cpu.trace(1, "INR M")
			cycle = cpu.INRM()
		case 0x35:
			cpu.trace(1, "DCR M")
			cycle = cpu.DCRM()
		case 0x01:
			cpu.trace(3, "LXI B, d16")
			cycle = cpu.LXIRPD16("b", "c")
		case 0x11:
			cpu.trace(3, "LXI D, d16")
			cycle = cpu.LXIRPD16("d", "e")
		case 0x21:
			cpu.trace(3, "LXI H, d16")
			cycle = cpu.LXIRPD16("h", "l")
		case 0x03:
			cpu.trace(1, "INX B")
			cycle = cpu.INXRP("bc")
		case 0x13:
			cpu.trace(1, "INX D")
			cycle = cpu.INXRP("de")
		case 0x23:
			cpu.trace(1, "INX H")
			cycle = cpu.INXRP("hl")
		case 0xB9:
			cpu.trace(1, "CMP (C)c")
			cycle = cpu.CMPR("c")
		case 0x0B:
			cpu.trace(1, "DCX B")
			cycle = cpu.DCXRP("bc")
		case 0x1B:
			cpu.trace(1, "DCX D")
			cycle = cpu.DCXRP("de")
		case 0x2B:
			cpu.trace(1, "DCX H")
			cycle = cpu.DCXRP("hl")
		case 0x32:
			cpu.trace(3, "STA addr")
			cycle = cpu.STAADDR()
		case 0x3A:
			cpu.trace(3, "LDA addr")
			cycle = cpu.LDAADDR()
		case 0x2A:
			cpu.trace(3, "LHLD addr")
			cycle = cpu.LHLDADDR()
		case 0x22:
			cpu.trace(3, "SHLD addr")
			cycle = cpu.SHLDADDR()
		case 0x0A:
			cpu.trace(1, "LDAX B")
			cycle = cpu.LDAXRP("bc")
		case 0x02:
			cpu.trace(1, "STAX B")
			cycle = cpu.STAXRP("bc")
		case 0xEB:
			cpu.trace(1, "XCHG")
			cycle = cpu.XCHG()
		case 0x1A:
			cpu.trace(1, "LDAX D")
			cycle = cpu.LDAXRP("de")
		case 0x12:
			cpu.trace(1, "STAX D")
			cycle = cpu.STAXRP("de")
		case 0x29:
			cpu.trace(1, "DAD H")
			cycle = cpu.DADRP("hl")
		case 0x09:
			cpu.trace(1, "DAD B")
			cycle = cpu.DADRP("bc")
		case 0x19:
			cpu.trace(1, "DAD D")
			cycle = cpu.DADRP("de")
		case 0x37:
			cpu.trace(1, "STC")
			cycle = cpu.STC()
		case 0x3F:
			cpu.trace(1, "CMC")
			cycle = cpu.CMC()
		case 0x2F:
			cpu.trace(1, "CMA")
			cycle = cpu.CMA()
		case 0x27:
			cpu.trace(1, "DAA")
			cycle = cpu.DAA()
		case 0x07:
			cpu.trace(1, "RLC")
			cycle = cpu.RLC()
		case 0x0F:
			cpu.trace(1, "RRC")
			cycle = cpu.RRC()
		case 0x17:
			cpu.trace(1, "RAL")
			cycle = cpu.RAL()
		case 0x1F:
			cpu.trace(1, "RAR")
			cycle = cpu.RAR()
		case 0xC5:
			cpu.trace(1, "PUSH B")
			cycle = cpu.PUSHRP("b", "c")
		case 0xD5:
			cpu.trace(1, "PUSH D")
			cycle = cpu.PUSHRP("d", "e")
		case 0xE5:
			cpu.trace(1, "PUSH H")
			cycle = cpu.PUSHRP("h", "l")
		case 0xF5:
			cpu.trace(1, "PUSH PSW")
			cycle = cpu.PUSHPSW()
		case 0xF1:
			cpu.trace(1, "POP PSW")
			cycle = cpu.POPPSW()
		case 0xE1:
			cpu.trace(1, "POP H")
			cycle = cpu.POPRP("h", "l")
		case 0xD1:
			cpu.trace(1, "POP D")
			cycle = cpu.POPRP("d", "e")
		case 0xC1:
			cpu.trace(1, "POP B")
			cycle = cpu.POPRP("b", "c")
		case 0x39:
			cpu.trace(1, "DAD (SP)sp")
			cycle = cpu.DADSP()
		case 0x3B:
			cpu.trace(1, "DCX (SP)sp")
			cycle = cpu.DCXSP()
		case 0x33:
			cpu.trace(1, "INX (SP)sp")
			cycle = cpu.INXSP()
		case 0xF9:
			cpu.trace(1, "SPHL")
			cycle = cpu.SPHL()
		case 0xE3:
			cpu.trace(1, "XTHL")
			cycle = cpu.XTHL()
		case 0xE9:
			cpu.trace(1, "PCHL")
			cycle = cpu.PCHL()
		case 0xC9:
			cpu.trace(1, "RET")
			cycle = cpu.RET()
		case 0x76:
			cpu.trace(1, "HLT")
			os.Exit(4)
		case 0xA0:
			cpu.trace(1, "ANA (B)b")
			cycle = cpu.ANAR("b")
		case 0x71:
			cpu.trace(1, "MOV M, (C)c")
			cycle = cpu.MOVMR("c")
		case 0x4E:
			cpu.trace(1, "MOV (C)c, M")
			cycle = cpu.MOVRM("c")
		case 0xFB:
			cpu.trace(1, "EI")
			cycle = cpu.EI()
		case 0xDB:
			cpu.trace(2, "IN")
			cycle = cpu.IN()
		case 0xD3:
			cpu.trace(2, "OUT")
			cycle = cpu.OUT()
		default:
			fmt.Printf("Unknown Opcode: %v, PC: %v\n", fmt.Sprintf("%X", cpu.opcode), fmt.Sprintf("%X", cpu.pc))
			//fmt.Println(fmt.Sprintf("%X", cpu.a), cpu.carry, cpu.parity, cpu.sign, cpu.zero)
			os.Exit(2)
	}

	return cycle
}