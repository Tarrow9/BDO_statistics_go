// main.go — Python 구현과 1:1 동작 일치(동률 비교는 '빈도만', MSB-first)
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

/*** ---------- 데이터 구조 ---------- ***/
type Node struct {
	c           byte
	f           uint32
	left, right *Node
}

type freqEntry struct {
	c byte
	f uint32
}

/*** ---------- MinHeap (파이썬과 동일: 빈도만 비교) ---------- ***/
type MinHeap struct {
	arr []*Node
}

func (h *MinHeap) size() int { return len(h.arr) }

func le(a, b *Node) bool { // a <= b  (빈도만)
	return a.f <= b.f
}
func lt(a, b *Node) bool { // a < b   (빈도만)
	return a.f < b.f
}

func (h *MinHeap) push(n *Node) {
	h.arr = append(h.arr, n)
	i := len(h.arr) - 1
	for {
		parent := (i - 1) / 2
		if le(h.arr[parent], h.arr[i]) { // parent <= child 이면 stop (파이썬과 동일)
			return
		}
		h.arr[parent], h.arr[i] = h.arr[i], h.arr[parent]
		i = parent
		if i <= 0 {
			return
		}
	}
}

func (h *MinHeap) pop() *Node {
	if h.size() == 0 {
		return nil
	}
	out := h.arr[0]
	last := h.arr[h.size()-1]
	h.arr = h.arr[:h.size()-1]
	if h.size() == 0 {
		return out
	}
	h.arr[0] = last

	parent := 0
	child := 2*parent + 1
	for child < h.size() {
		if child+1 < h.size() && lt(h.arr[child+1], h.arr[child]) { // 더 작은 자식(빈도만 비교)
			child++
		}
		if le(h.arr[parent], h.arr[child]) { // parent <= child 이면 stop
			return out
		}
		h.arr[parent], h.arr[child] = h.arr[child], h.arr[parent]
		parent = child
		child = 2*child + 1
	}
	return out
}

/*** ---------- 트리 구성 (입력 순서 보존해 push) ---------- ***/
func makeTreeOrdered(entries []freqEntry) *Node {
	h := &MinHeap{}
	for _, e := range entries { // 파일에서 읽힌 순서대로 push (파이썬 dict의 insertion order와 일치)
		h.push(&Node{c: e.c, f: e.f})
	}
	for h.size() > 1 {
		a := h.pop()
		b := h.pop()
		h.push(&Node{f: a.f + b.f, left: a, right: b}) // a=left, b=right
	}
	return h.pop()
}

/*** ---------- MSB-first 비트 리더 ---------- ***/
type bitReader struct {
	data []byte
	bits int
	pos  int
}

func newBitReader(b []byte, bits int) *bitReader { return &bitReader{data: b, bits: bits} }

func (br *bitReader) readBit() (bool, error) {
	if br.pos >= br.bits {
		return false, io.EOF
	}
	byteIdx := br.pos / 8
	bitOff := br.pos % 8
	v := (br.data[byteIdx]>>(7-uint(bitOff)))&1 == 1 // MSB-first (BitArray와 동일)
	br.pos++
	return v, nil
}

/*** ---------- 디코딩 ---------- ***/
func decode(tree *Node, packed []byte, bitCount int) (string, error) {
	if tree == nil {
		return "", errors.New("invalid tree: empty")
	}
	br := newBitReader(packed, bitCount)
	out := make([]byte, 0, 1024)

	for br.pos < br.bits {
		n := tree
		for {
			if n.left == nil && n.right == nil {
				break
			}
			bit, err := br.readBit()
			if err != nil {
				return "", fmt.Errorf("invalid tree/bitstream: %w (unpacked=%q)", err, string(out))
			}
			if bit {
				n = n.right
			} else {
				n = n.left
			}
			if n == nil {
				return "", fmt.Errorf("invalid tree: dead end (unpacked=%q)", string(out))
			}
		}
		out = append(out, n.c)
	}
	return string(out), nil
}

/*** ---------- 리틀엔디언 uint32 ---------- ***/
func readU32(r io.Reader) (uint32, error) {
	var v uint32
	if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
		return 0, err
	}
	return v, nil
}

/*** ---------- 헤더 파싱 (정확히 파이썬과 동일: 3개 길이 필드) ---------- ***/
func getFreqsOrdered(r io.Reader) ([]freqEntry, error) {
	// file_len, always0, chars_count
	if _, err := readU32(r); err != nil { // file_len
		return nil, err
	}
	if _, err := readU32(r); err != nil { // always0
		return nil, err
	}
	chars, err := readU32(r) // chars_count
	if err != nil {
		return nil, err
	}

	entries := make([]freqEntry, 0, chars)
	for i := uint32(0); i < chars; i++ {
		cnt, err := readU32(r) // count
		if err != nil {
			return nil, err
		}
		var c [1]byte
		if _, err := io.ReadFull(r, c[:]); err != nil { // 'cxxx'의 'c'
			return nil, err
		}
		var pad [3]byte
		if _, err := io.ReadFull(r, pad[:]); err != nil { // 'cxxx'의 'xxx'
			return nil, err
		}
		entries = append(entries, freqEntry{c: c[0], f: cnt})
	}
	return entries, nil
}

/*** ---------- 공개 API ---------- ***/
func UnpackFromReader(r io.Reader) (string, error) {
	entries, err := getFreqsOrdered(r)
	if err != nil {
		return "", err
	}
	tree := makeTreeOrdered(entries)
	if tree == nil {
		return "", errors.New("empty frequency table")
	}

	// 파이썬과 동일하게 정확히 3개만 읽어요.
	packedBits, err := readU32(r)
	if err != nil {
		return "", err
	}
	packedBytes, err := readU32(r)
	if err != nil {
		return "", err
	}
	// unpackedBytes는 읽기만 (검증/정보용)
	_, err = readU32(r)
	if err != nil {
		return "", err
	}

	packed := make([]byte, packedBytes)
	if _, err := io.ReadFull(r, packed); err != nil {
		return "", err
	}
	return decode(tree, packed, int(packedBits))
}

func UnpackBytes(b []byte) (string, error) { return UnpackFromReader(bytes.NewReader(b)) }

/*** ---------- 데모 ---------- ***/
func main() {
	// 인자가 있으면 파일에서 읽고, 없으면 데모 벡터 사용해요.
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "파일 열기 실패:", err)
			os.Exit(1)
		}
		defer f.Close()
		out, err := UnpackFromReader(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "언팩 실패:", err)
			os.Exit(1)
		}
		fmt.Println(out)
		return
	}

	// 제공하신 데모 바이트 (여기서 0x7C는 마지막 문자 '|'의 cxxx임)
	test := []byte{
		0x81, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x0B, 0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00,
		0x2D, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00,
		0x30, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		0x31, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		0x32, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00,
		0x33, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00,
		0x34, 0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00,
		0x35, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		0x37, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00,
		0x38, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		0x39, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, // <- 여기까지 11개 빈도 (마지막이 '|')
		0x7C, 0x00, 0x00, 0x00, // '|'의 cxxx (추가 필드가 아님)
		0x85, 0x00, 0x00, 0x00, // packedBits = 133
		0x11, 0x00, 0x00, 0x00, // packedBytes = 17
		0x29, 0x00, 0x00, 0x00, // unpackedBytes = 41
		0xD3, 0x0C, 0x78, 0x90, 0xFB, 0x1D, 0x0E, 0x6E,
		0x4B, 0x4C, 0x35, 0xDF, 0x17, 0x75, 0xBD, 0xAA, 0x90,
	}

	out, err := UnpackBytes(test)
	if err != nil {
		fmt.Fprintln(os.Stderr, "데모 언팩 실패:", err)
		os.Exit(1)
	}
	fmt.Println(out) // 기대: 53801-198-55428-4050|53802-0-17725-70000|
}
