package hash

import (
    "encoding/binary"
)

func Murmur3_32(key string, seed uint32) uint32 {
    data := []byte(key)
    length := len(data)
    nblocks := length / 4

    h1 := seed

    const c1 uint32 = 0xcc9e2d51
    const c2 uint32 = 0x1b873593

    for i := 0; i < nblocks; i++ {
        k1 := binary.LittleEndian.Uint32(data[i*4:])
        k1 *= c1
        k1 = (k1 << 15) | (k1 >> 17)
        k1 *= c2
        h1 ^= k1
        h1 = (h1 << 13) | (h1 >> 19)
        h1 = h1*5 + 0xe6546b64
    }

    tail := data[nblocks*4:]
    var k1 uint32 = 0
    switch length & 3 {
    case 3:
        k1 ^= uint32(tail[2]) << 16
        fallthrough
    case 2:
        k1 ^= uint32(tail[1]) << 8
        fallthrough
    case 1:
        k1 ^= uint32(tail[0])
        k1 *= c1
        k1 = (k1 << 15) | (k1 >> 17)
        k1 *= c2
        h1 ^= k1
    }

    h1 ^= uint32(length)
    h1 ^= h1 >> 16
    h1 *= 0x85ebca6b
    h1 ^= h1 >> 13
    h1 *= 0xc2b2ae35
    h1 ^= h1 >> 16

    return h1
}