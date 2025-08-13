package huffmanunpack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// Node는 허프만 트리의 노드를 나타내요.
type Node struct {
	c           byte
	f           uint32
	left, right *Node
}

// ===== Python MinHeap을 그대로 재현한 힙 =====

type MinHeap struct {
	arr []*Node
}

func (h *MinHeap) size() int { return len(h.arr) }

func le(a, b *Node) bool { // a <= b
	return a.f <= b.f
}
func lt(a, b *Node) bool { // a < b
	return a.f < b.f
}

func (h *MinHeap) push(n *Node) {
	h.arr = append(h.arr, n)
	childIdx := h.size() - 1

	for {
		if childIdx <= 0 {
			return
		}
		parentIdx := (childIdx - 1) / 2
		if le(h.arr[parentIdx], h.arr[childIdx]) {
			return
		}
		h.arr[parentIdx], h.arr[childIdx] = h.arr[childIdx], h.arr[parentIdx]
		childIdx = parentIdx
		if childIdx <= 0 {
			return
		}
	}
}

func (h *MinHeap) pop() *Node {
	if h.size() == 0 {
		return nil
	}
	obj := h.arr[0]
	last := h.arr[h.size()-1]
	h.arr = h.arr[:h.size()-1]
	if h.size() == 0 {
		return obj
	}
	h.arr[0] = last

	parentIdx := 0
	childIdx := 2*parentIdx + 1
	for childIdx < h.size() {
		if childIdx+1 < h.size() {
			if lt(h.arr[childIdx+1], h.arr[childIdx]) {
				childIdx++
			}
		}
		if le(h.arr[parentIdx], h.arr[childIdx]) {
			return obj
		}
		h.arr[parentIdx], h.arr[childIdx] = h.arr[childIdx], h.arr[parentIdx]
		parentIdx = childIdx
		childIdx = 2*childIdx + 1
	}
	return obj
}

// ===== 트리 구성 =====

func makeTree(freqs map[byte]uint32) *Node {
	h := &MinHeap{}
	for c, f := range freqs {
		h.push(&Node{c: c, f: f})
	}
	for h.size() > 1 {
		a := h.pop()
		b := h.pop()
		n := &Node{f: a.f + b.f, left: a, right: b}
		h.push(n)
	}
	return h.pop()
}

// ===== 비트 읽기(MSB-first) =====

type bitReader struct {
	data []byte
	bits int
	pos  int // 읽은 비트 수
}

func newBitReader(p []byte, bits int) *bitReader {
	return &bitReader{data: p, bits: bits, pos: 0}
}

func (br *bitReader) readBit() (bool, error) {
	if br.pos >= br.bits {
		return false, io.EOF
	}
	byteIdx := br.pos / 8
	bitOff := br.pos % 8 // MSB-first
	b := br.data[byteIdx]
	bit := (b>>(7-uint(bitOff)))&1 == 1
	br.pos++
	return bit, nil
}

// ===== 디코딩 =====

func decode(tree *Node, freqs map[byte]uint32, packed []byte, bitCount int) (string, error) {
	if tree == nil {
		return "", errors.New("invalid tree: empty")
	}
	br := newBitReader(packed, bitCount)
	out := make([]byte, 0, 1024)

	for br.pos < br.bits {
		node := tree
		for {
			// 리프면 종료
			if node.left == nil && node.right == nil {
				break
			}
			// 다음 비트
			bit, err := br.readBit()
			if err == io.EOF {
				return "", fmt.Errorf("invalid tree: out of message bounds, unpacked=%q", string(out))
			}
			if err != nil {
				return "", err
			}
			if bit {
				node = node.right
			} else {
				node = node.left
			}
			if node == nil {
				return "", fmt.Errorf("invalid tree: dead end while walking, unpacked=%q", string(out))
			}
		}
		out = append(out, node.c)
	}
	return string(out), nil
}

// ===== 리틀엔디언 uint32 읽기 =====

func readU32(r io.Reader) (uint32, error) {
	var v uint32
	if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
		return 0, err
	}
	return v, nil
}

// ===== 헤더 파싱 =====

func getFreqs(r io.Reader) (map[byte]uint32, error) {
	// file_len, always0, chars_count
	if _, err := readU32(r); err != nil { // file_len (미사용)
		return nil, err
	}
	if _, err := readU32(r); err != nil { // always0 (미사용)
		return nil, err
	}
	charsCount, err := readU32(r)
	if err != nil {
		return nil, err
	}
	freqs := make(map[byte]uint32, charsCount)
	for i := uint32(0); i < charsCount; i++ {
		count, err := readU32(r)
		if err != nil {
			return nil, err
		}
		var c [1]byte
		if _, err := io.ReadFull(r, c[:]); err != nil {
			return nil, err
		}
		// padding 3 bytes (cxxx)
		var pad [3]byte
		if _, err := io.ReadFull(r, pad[:]); err != nil {
			return nil, err
		}
		freqs[c[0]] = count
	}
	return freqs, nil
}

// ===== 언팩 =====

func unpackFromReader(r io.Reader) (string, error) {
	freqs, err := getFreqs(r)
	if err != nil {
		return "", err
	}
	tree := makeTree(freqs)
	if tree == nil {
		return "", errors.New("empty frequency table")
	}

	packedBits, err := readU32(r)
	if err != nil {
		return "", err
	}
	packedBytes, err := readU32(r)
	if err != nil {
		return "", err
	}
	// unpackedBytes는 확인용으로만 읽고 사용하지 않아요(파이썬 코드와 동일)
	_, err = readU32(r)
	if err != nil {
		return "", err
	}

	packed := make([]byte, packedBytes)
	if _, err := io.ReadFull(r, packed); err != nil {
		return "", err
	}

	return decode(tree, freqs, packed, int(packedBits))
}

func unpackBytes(data []byte) (string, error) {
	return unpackFromReader(bytes.NewReader(data))
}

// ===== 데모 & CLI =====

func main() {
	// 사용법:
	//   go run huffman_unpack.go <파일경로>
	// 인자가 없으면 파이썬 예제 벡터로 데모를 실행해요.
	if len(os.Args) > 1 {
		path := os.Args[1]
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "파일 열기 실패: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out, err := unpackFromReader(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "언팩 실패: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(out)
		return
	}

	// 파이썬 코드의 테스트 벡터
	test := []byte{
		0x81, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0B, 0x00, 0x00, 0x00,
		0x06, 0x00, 0x00, 0x00, 0x2D, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00,
		0x30, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x31, 0x00, 0x00, 0x00,
		0x03, 0x00, 0x00, 0x00, 0x32, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00,
		0x33, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x34, 0x00, 0x00, 0x00,
		0x06, 0x00, 0x00, 0x00, 0x35, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		0x37, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x38, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00, 0x39, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00,
		0x7C, 0x00, 0x00, 0x00, 0x85, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00,
		0x29, 0x00, 0x00, 0x00, 0xD3, 0x0C, 0x78, 0x90, 0xFB, 0x1D, 0x0E, 0x6E,
		0x4B, 0x4C, 0x35, 0xDF, 0x17, 0x75, 0xBD, 0xAA, 0x90,
	}
	out, err := unpackBytes(test)
	if err != nil {
		fmt.Fprintf(os.Stderr, "데모 언팩 실패: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(out)
}
