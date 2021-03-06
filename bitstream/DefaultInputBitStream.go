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

package bitstream

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type DefaultInputBitStream struct {
	closed      bool
	read        uint64
	position    int // index of current byte (consumed if bitIndex == -1)
	availBits   int // bits not consumed in current
	is          io.ReadCloser
	buffer      []byte
	maxPosition int
	current     uint64 // cached bits
}

func NewDefaultInputBitStream(stream io.ReadCloser, bufferSize uint) (*DefaultInputBitStream, error) {
	if stream == nil {
		return nil, errors.New("Invalid null input stream parameter")
	}

	if bufferSize < 1024 {
		return nil, errors.New("Invalid buffer size parameter (must be at least 1024 bytes)")
	}

	if bufferSize > 1<<29 {
		return nil, errors.New("Invalid buffer size parameter (must be at most 536870912 bytes)")
	}

	if bufferSize&7 != 0 {
		return nil, errors.New("Invalid buffer size (must be a multiple of 8)")
	}

	this := new(DefaultInputBitStream)
	this.buffer = make([]byte, bufferSize)
	this.is = stream
	this.availBits = 0
	this.maxPosition = -1
	return this, nil
}

// Return 1 or 0
func (this *DefaultInputBitStream) ReadBit() int {
	if this.availBits == 0 {
		this.pullCurrent() // Panic if stream is closed
	}

	this.availBits--
	return int(this.current>>uint(this.availBits)) & 1
}

func (this *DefaultInputBitStream) ReadBits(count uint) uint64 {
	if count == 0 || count > 64 {
		panic(fmt.Errorf("Invalid count: %v (must be in [1..64])", count))
	}

	var res uint64
	remaining := int(count) - this.availBits

	if remaining <= 0 {
		// Enough spots available in 'current'
		if this.availBits == 0 {
			this.pullCurrent()
			remaining -= (this.availBits - 64) // adjust if availBits != 64 (end of stream)
		}

		res = (this.current >> uint(-remaining)) & (0xFFFFFFFFFFFFFFFF >> (64 - count))
		this.availBits -= int(count)
	} else {
		// Not enough spots available in 'current'
		res = this.current & ((uint64(1) << uint(this.availBits)) - 1)
		this.pullCurrent()
		this.availBits -= remaining
		res = (res << uint(remaining)) | (this.current >> uint(this.availBits))
	}

	return res
}

func (this *DefaultInputBitStream) ReadArray(bits []byte, count uint) uint {
	if this.Closed() {
		panic(errors.New("Stream closed"))
	}

	if count == 0 {
		return 0
	}

	remaining := int(count)
	start := 0

	// Byte aligned cursor ?
	if this.availBits&7 == 0 {
		if this.availBits == 0 {
			this.pullCurrent()
		}

		// Empty this.current
		for this.availBits > 0 && remaining >= 8 {
			bits[start] = byte(this.ReadBits(8))
			start++
			remaining -= 8
		}

		// Copy internal buffer to bits array
		for (remaining >> 3) > this.maxPosition+1-this.position {
			copy(bits[start:], this.buffer[this.position:this.maxPosition+1])
			start += (this.maxPosition + 1 - this.position)
			remaining -= ((this.maxPosition + 1 - this.position) << 3)

			if _, err := this.readFromInputStream(len(this.buffer)); err != nil {
				panic(err)
			}
		}

		r := (remaining >> 6) << 3

		if r > 0 {
			copy(bits[start:start+r], this.buffer[this.position:this.position+r])
			this.position += r
			start += r
			remaining -= (r << 3)
		}
	} else {
		// Not byte aligned
		r := 64 - this.availBits

		for remaining >= 64 {
			v := this.current & ((uint64(1) << uint(this.availBits)) - 1)
			this.pullCurrent()
			this.availBits -= r
			binary.BigEndian.PutUint64(bits[start:start+8], (v<<uint(r))|(this.current>>uint(this.availBits)))
			start += 8
			remaining -= 64
		}
	}

	// Last bytes
	for remaining >= 8 {
		bits[start] = byte(this.ReadBits(8))
		start++
		remaining -= 8
	}

	if remaining > 0 {
		bits[start] = byte(this.ReadBits(uint(remaining)) << uint(8-remaining))
	}

	return count
}

func (this *DefaultInputBitStream) readFromInputStream(count int) (int, error) {
	if this.Closed() {
		return 0, errors.New("Stream closed")
	}

	this.read += uint64((this.maxPosition + 1) << 3)
	size, err := this.is.Read(this.buffer[0:count])
	this.position = 0

	if size <= 0 {
		this.maxPosition = -1
	} else {
		this.maxPosition = size - 1
	}

	if err != nil {
		return size, err
	}

	if size <= 0 {
		return size, errors.New("No more data to read in the bitstream")
	}

	return size, nil
}

func (this *DefaultInputBitStream) HasMoreToRead() (bool, error) {
	if this.Closed() {
		return false, errors.New("Stream closed")
	}

	if this.position < this.maxPosition || this.availBits > 0 {
		return true, nil
	}

	_, err := this.readFromInputStream(len(this.buffer))
	return err == nil, err
}

// Pull 64 bits of current value from buffer.
func (this *DefaultInputBitStream) pullCurrent() {
	if this.position > this.maxPosition {
		if _, err := this.readFromInputStream(len(this.buffer)); err != nil {
			panic(err)
		}
	}

	if this.position+7 > this.maxPosition {
		// End of stream: overshoot max position => adjust bit index
		shift := uint(this.maxPosition-this.position) << 3
		this.availBits = int(shift) + 8
		val := uint64(0)

		for this.position <= this.maxPosition {
			val |= (uint64(this.buffer[this.position]&0xFF) << shift)
			this.position++
			shift -= 8
		}

		this.current = val
	} else {
		// Regular processing, buffer length is multiple of 8
		this.current = binary.BigEndian.Uint64(this.buffer[this.position : this.position+8])
		this.availBits = 64
		this.position += 8
	}

}

func (this *DefaultInputBitStream) Close() (bool, error) {
	if this.Closed() {
		return true, nil
	}

	this.closed = true

	// Reset fields to force a readFromInputStream() and trigger an error
	// on ReadBit() or ReadBits()
	this.availBits = 0
	this.maxPosition = -1
	return true, nil
}

// Return number of bits read so far
func (this *DefaultInputBitStream) Read() uint64 {
	return this.read + uint64(this.position)<<3 - uint64(this.availBits)
}

func (this *DefaultInputBitStream) Closed() bool {
	return this.closed
}
