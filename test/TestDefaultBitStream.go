/*
Copyright 2011-2017 Frederic Langlet
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
you may obtain a copy of the License at

                http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	kanzi "github.com/flanglet/kanzi-go"
	"github.com/flanglet/kanzi-go/bitstream"
	"github.com/flanglet/kanzi-go/io"
	"math/rand"
	"os"
	"time"
)

func main() {
	testCorrectnessAligned1()
	testCorrectnessAligned2()
	testCorrectnessMisaligned1()
	testCorrectnessMisaligned2()
	var filename = flag.String("filename", "r:\\output.bin", "Ouput file name for speed test")
	flag.Parse()
	testSpeed1(filename) // Writes big output.bin file to local dir !!!
	testSpeed2(filename) // Writes big output.bin file to local dir !!!
}

func testCorrectnessAligned1() {
	fmt.Printf("Correctness Test - write long - byte aligned\n")
	values := make([]int, 100)
	rand.Seed(time.Now().UTC().UnixNano())

	// Check correctness of read() and written()
	for t := 1; t <= 32; t++ {
		var bs io.BufferStream
		obs, _ := bitstream.NewDefaultOutputBitStream(&bs, 16384)
		fmt.Println()
		obs.WriteBits(0x0123456789ABCDEF, uint(t))
		fmt.Printf("Written (before close): %v\n", obs.Written())
		obs.Close()
		fmt.Printf("Written (after close): %v\n", obs.Written())

		ibs, _ := bitstream.NewDefaultInputBitStream(&bs, 16384)
		ibs.ReadBits(uint(t))

		if ibs.Read() == uint64(t) {
			fmt.Println("OK")
		} else {
			fmt.Println("KO")
		}

		fmt.Printf("Read (before close): %v\n", ibs.Read())
		ibs.Close()
		fmt.Printf("Read (after close): %v\n", ibs.Read())
	}

	for test := 1; test <= 10; test++ {
		var bs io.BufferStream
		obs, _ := bitstream.NewDefaultOutputBitStream(&bs, 16384)
		dbgbs, _ := bitstream.NewDebugOutputBitStream(obs, os.Stdout)
		dbgbs.ShowByte(true)
		dbgbs.Mark(true)

		for i := range values {
			if test < 5 {
				values[i] = rand.Intn(test*1000 + 100)
			} else {
				values[i] = rand.Intn(1 << 31)
			}

			fmt.Printf("%v ", values[i])

			if i%20 == 19 {
				println()
			}
		}

		println()
		println()

		for i := range values {
			dbgbs.WriteBits(uint64(values[i]), 32)
		}

		// Close first to force flush()
		dbgbs.Close()

		ibs, _ := bitstream.NewDefaultInputBitStream(&bs, 16384)
		fmt.Printf("\nRead:\n")
		ok := true

		for i := range values {
			x := ibs.ReadBits(32)
			fmt.Printf("%v", x)

			if int(x) == values[i] {
				fmt.Printf(" ")
			} else {
				fmt.Printf("* ")
				ok = false
			}

			if i%20 == 19 {
				println()
			}
		}

		ibs.Close()
		bs.Close()
		println()
		println()
		fmt.Printf("Bits written: %v\n", dbgbs.Written())
		fmt.Printf("Bits read: %v\n", ibs.Read())

		if ok {
			fmt.Printf("\nSuccess\n")
		} else {
			fmt.Printf("\nFailure\n")
		}

		println()
		println()
	}

}

func testCorrectnessMisaligned1() {
	fmt.Printf("Correctness Test - write long - not byte aligned\n")
	values := make([]int, 100)
	rand.Seed(time.Now().UTC().UnixNano())

	// Check correctness of read() and written()
	for t := 1; t <= 32; t++ {
		var bs io.BufferStream
		obs, _ := bitstream.NewDefaultOutputBitStream(&bs, 16384)
		fmt.Println()
		obs.WriteBit(1)
		obs.WriteBits(0x0123456789ABCDEF, uint(t))
		fmt.Printf("Written (before close): %v\n", obs.Written())
		obs.Close()
		fmt.Printf("Written (after close): %v\n", obs.Written())

		ibs, _ := bitstream.NewDefaultInputBitStream(&bs, 16384)
		ibs.ReadBit()
		ibs.ReadBits(uint(t))

		if ibs.Read() == uint64(t+1) {
			fmt.Println("OK")
		} else {
			fmt.Println("KO")
		}
	}

	for test := 1; test <= 10; test++ {
		var bs io.BufferStream
		obs, _ := bitstream.NewDefaultOutputBitStream(&bs, 16384)
		dbgbs, _ := bitstream.NewDebugOutputBitStream(obs, os.Stdout)
		dbgbs.ShowByte(true)
		dbgbs.Mark(true)

		for i := range values {
			if test < 5 {
				values[i] = rand.Intn(test*1000 + 100)
			} else {
				values[i] = rand.Intn(1 << 31)
			}

			mask := (1 << (1 + uint(i&63))) - 1
			values[i] &= mask
			fmt.Printf("%v ", values[i])

			if i%20 == 19 {
				println()
			}
		}

		println()
		println()

		for i := range values {
			dbgbs.WriteBits(uint64(values[i]), 1+uint(i&63))
		}

		// Close first to force flush()
		dbgbs.Close()
		testWritePostClose(dbgbs)

		ibs, _ := bitstream.NewDefaultInputBitStream(&bs, 16384)
		fmt.Printf("\nRead:\n")
		ok := true

		for i := range values {
			x := ibs.ReadBits(1 + uint(i&63))
			fmt.Printf("%v", x)

			if int(x) == values[i] {
				fmt.Printf(" ")
			} else {
				fmt.Printf("* ")
				ok = false
			}

			if i%20 == 19 {
				println()
			}
		}

		ibs.Close()
		testReadPostClose(ibs)
		bs.Close()

		println()
		println()
		fmt.Printf("Bits written: %v\n", dbgbs.Written())
		fmt.Printf("Bits read: %v\n", ibs.Read())

		if ok {
			fmt.Printf("\nSuccess\n")
		} else {
			fmt.Printf("\nFailure\n")
		}

		println()
		println()
	}
}
func testCorrectnessAligned2() {
	fmt.Printf("Correctness Test - write array - byte aligned\n")
	input := make([]byte, 100)
	output := make([]byte, 100)
	rand.Seed(time.Now().UTC().UnixNano())

	for test := 1; test <= 10; test++ {
		var bs io.BufferStream
		obs, _ := bitstream.NewDefaultOutputBitStream(&bs, 16384)
		dbgbs, _ := bitstream.NewDebugOutputBitStream(obs, os.Stdout)
		dbgbs.ShowByte(true)
		dbgbs.Mark(true)
		println()

		for i := range input {
			if test < 5 {
				input[i] = byte(rand.Intn(test*1000 + 100))
			} else {
				input[i] = byte(rand.Intn(1 << 31))
			}

			fmt.Printf("%v ", input[i])

			if i%20 == 19 {
				println()
			}
		}

		count := uint(8 + test*(20+(test&1)) + (test & 3))
		println()
		println()
		dbgbs.WriteArray(input, count)

		// Close first to force flush()
		dbgbs.Close()

		ibs, _ := bitstream.NewDefaultInputBitStream(&bs, 16384)
		fmt.Printf("\nRead:\n")
		r := ibs.ReadArray(output, count)
		ok := r == count

		if ok == true {
			for i := 0; i < int(r>>3); i++ {
				fmt.Printf("%v", output[i])

				if output[i] == input[i] {
					fmt.Printf(" ")
				} else {
					fmt.Printf("* ")
					ok = false
				}

				if i%20 == 19 {
					println()
				}
			}
		}

		ibs.Close()
		bs.Close()
		println()
		println()
		fmt.Printf("Bits written: %v\n", dbgbs.Written())
		fmt.Printf("Bits read: %v\n", ibs.Read())

		if ok {
			fmt.Printf("\nSuccess\n")
		} else {
			fmt.Printf("\nFailure\n")
		}

		println()
		println()
	}

}

func testCorrectnessMisaligned2() {
	fmt.Printf("Correctness Test - write array - not byte aligned\n")
	input := make([]byte, 100)
	output := make([]byte, 100)
	rand.Seed(time.Now().UTC().UnixNano())

	for test := 1; test <= 10; test++ {
		var bs io.BufferStream
		obs, _ := bitstream.NewDefaultOutputBitStream(&bs, 16384)
		dbgbs, _ := bitstream.NewDebugOutputBitStream(obs, os.Stdout)
		dbgbs.ShowByte(true)
		dbgbs.Mark(true)
		println()

		for i := range input {
			if test < 5 {
				input[i] = byte(rand.Intn(test*1000 + 100))
			} else {
				input[i] = byte(rand.Intn(1 << 31))
			}

			fmt.Printf("%v ", input[i])

			if i%20 == 19 {
				println()
			}
		}

		count := uint(8 + test*(20+(test&1)) + (test & 3))
		println()
		println()
		dbgbs.WriteBit(0)
		dbgbs.WriteArray(input[1:], count)

		// Close first to force flush()
		dbgbs.Close()

		ibs, _ := bitstream.NewDefaultInputBitStream(&bs, 16384)
		fmt.Printf("\nRead:\n")
		ibs.ReadBit()
		r := ibs.ReadArray(output[1:], count)
		ok := r == count

		if ok == true {
			for i := 1; i < 1+int(r>>3); i++ {
				fmt.Printf("%v", output[i])

				if output[i] == input[i] {
					fmt.Printf(" ")
				} else {
					fmt.Printf("* ")
					ok = false
				}

				if i%20 == 19 {
					println()
				}
			}
		}

		ibs.Close()
		bs.Close()
		println()
		println()
		fmt.Printf("Bits written: %v\n", dbgbs.Written())
		fmt.Printf("Bits read: %v\n", ibs.Read())

		if ok {
			fmt.Printf("\nSuccess\n")
		} else {
			fmt.Printf("\nFailure\n")
		}

		println()
		println()
	}
}

func testWritePostClose(obs kanzi.OutputBitStream) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Error: %v\n", r.(error).Error())
		}
	}()

	fmt.Printf("\nTrying to write to closed stream\n")
	obs.WriteBit(1)
}

func testReadPostClose(ibs kanzi.InputBitStream) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Error: %v\n", r.(error).Error())
		}
	}()

	fmt.Printf("\nTrying to read from closed stream\n")
	ibs.ReadBit()
}

func testSpeed1(filename *string) {
	fmt.Printf("Speed Test 1\n")

	values := []uint64{3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5, 8, 9, 7, 9, 3,
		31, 14, 41, 15, 59, 92, 26, 65, 53, 35, 58, 89, 97, 79, 93, 32}
	iter := 150
	read := uint64(0)
	written := uint64(0)
	delta1 := int64(0)
	delta2 := int64(0)
	nn := 100000 * len(values)
	defer os.Remove(*filename)

	for test := 0; test < iter; test++ {
		file1, err := os.Create(*filename)

		if err != nil {
			fmt.Printf("Cannot create %s", *filename)

			return
		}

		bos := file1
		obs, _ := bitstream.NewDefaultOutputBitStream(bos, 16*1024)
		before := time.Now()
		for i := 0; i < nn; i++ {
			obs.WriteBits(values[i%len(values)], 1+uint(i&63))
		}

		// Close first to force flush()
		obs.Close()
		delta1 += time.Now().Sub(before).Nanoseconds()
		written += obs.Written()
		file1.Close()

		file2, err := os.Open(*filename)

		if err != nil {
			fmt.Printf("Cannot open %s", *filename)

			return
		}

		bis := file2
		ibs, _ := bitstream.NewDefaultInputBitStream(bis, 1024*1024)
		before = time.Now()

		for i := 0; i < nn; i++ {
			ibs.ReadBits(1 + uint(i&63))
		}

		ibs.Close()
		delta2 += time.Now().Sub(before).Nanoseconds()
		read += ibs.Read()
		file2.Close()
	}

	println()
	fmt.Printf("%v bits written (%v MB)\n", written, written/1024/8192)
	fmt.Printf("%v bits read (%v MB)\n", read, read/1024/8192)
	println()
	fmt.Printf("Write [ms]        : %v\n", delta1/1000000)
	fmt.Printf("Throughput [MB/s] : %d\n", (written/1024*1000/8192)/uint64(delta1/1000000))
	fmt.Printf("Read [ms]         : %v\n", delta2/1000000)
	fmt.Printf("Throughput [MB/s] : %d\n", (read/1024*1000/8192)/uint64(delta2/1000000))
	println()
	println()
}

func testSpeed2(filename *string) {
	fmt.Printf("Speed Test2\n")

	values := []byte{3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5, 8, 9, 7, 9, 3,
		31, 14, 41, 15, 59, 92, 26, 65, 53, 35, 58, 89, 97, 79, 93, 32}
	iter := 150
	read := uint64(0)
	written := uint64(0)
	delta1 := int64(0)
	delta2 := int64(0)
	nn := 3250000 * len(values)
	defer os.Remove(*filename)
	input := make([]byte, nn)
	output := make([]byte, nn)

	for i := 0; i < 3250000; i++ {
		copy(input[i*len(values):], values)
	}

	for test := 0; test < iter; test++ {
		file1, err := os.Create(*filename)

		if err != nil {
			fmt.Printf("Cannot create %s", *filename)

			return
		}

		bos := file1
		obs, _ := bitstream.NewDefaultOutputBitStream(bos, 16*1024)
		before := time.Now()
		obs.WriteArray(input, uint(len(input)))

		// Close first to force flush()
		obs.Close()
		delta1 += time.Now().Sub(before).Nanoseconds()
		written += obs.Written()
		file1.Close()

		file2, err := os.Open(*filename)

		if err != nil {
			fmt.Printf("Cannot open %s", *filename)

			return
		}

		bis := file2
		ibs, _ := bitstream.NewDefaultInputBitStream(bis, 1024*1024)
		before = time.Now()
		ibs.ReadArray(output, uint(len(output)))

		ibs.Close()
		delta2 += time.Now().Sub(before).Nanoseconds()
		read += ibs.Read()
		file2.Close()
	}

	println()
	fmt.Printf("%v bits written (%v MB)\n", written, written/1024/8192)
	fmt.Printf("%v bits read (%v MB)\n", read, read/1024/8192)
	println()
	fmt.Printf("Write [ms]        : %v\n", delta1/1000000)
	fmt.Printf("Throughput [MB/s] : %d\n", (written/1024*1000/8192)/uint64(delta1/1000000))
	fmt.Printf("Read [ms]         : %v\n", delta2/1000000)
	fmt.Printf("Throughput [MB/s] : %d\n", (read/1024*1000/8192)/uint64(delta2/1000000))
	println()
	println()
}
