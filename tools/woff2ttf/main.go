// Command woff2ttf unpacks a WOFF 1.0 font into the TrueType file it wraps.
// The web UI's fonts ship as WOFF (web/node_modules/@fontsource); the Word
// and PDF exports embed the same faces as TrueType so they match the screen.
//
//	go run ./tools/woff2ttf in.woff out.ttf
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: woff2ttf in.woff out.ttf")
		os.Exit(2)
	}
	in, err := os.ReadFile(os.Args[1])
	if err == nil {
		var out []byte
		if out, err = unwrap(in); err == nil {
			err = os.WriteFile(os.Args[2], out, 0o644)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type table struct {
	tag      [4]byte
	checksum uint32
	data     []byte
}

func unwrap(in []byte) ([]byte, error) {
	if len(in) < 44 || string(in[:4]) != "wOFF" {
		return nil, errors.New("not a WOFF 1.0 file")
	}
	be := binary.BigEndian
	flavor := be.Uint32(in[4:])
	n := int(be.Uint16(in[12:]))
	if len(in) < 44+20*n {
		return nil, errors.New("truncated table directory")
	}
	tables := make([]table, 0, n)
	for i := 0; i < n; i++ {
		e := in[44+20*i:]
		off, comp, orig := be.Uint32(e[4:]), be.Uint32(e[8:]), be.Uint32(e[12:])
		if uint64(off)+uint64(comp) > uint64(len(in)) || comp > orig {
			return nil, errors.New("table outside the file")
		}
		data := in[off : off+comp]
		if comp < orig {
			r, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			if data, err = io.ReadAll(io.LimitReader(r, int64(orig)+1)); err != nil {
				return nil, err
			}
			if uint32(len(data)) != orig {
				return nil, errors.New("table length mismatch")
			}
		}
		var t table
		copy(t.tag[:], e[:4])
		t.checksum, t.data = be.Uint32(e[16:]), data
		tables = append(tables, t)
	}
	sort.Slice(tables, func(i, j int) bool { return bytes.Compare(tables[i].tag[:], tables[j].tag[:]) < 0 })
	pow, shift := 1, 0
	for pow*2 <= n {
		pow, shift = pow*2, shift+1
	}
	var out bytes.Buffer
	head := make([]byte, 12)
	be.PutUint32(head, flavor)
	be.PutUint16(head[4:], uint16(n))
	be.PutUint16(head[6:], uint16(pow*16))
	be.PutUint16(head[8:], uint16(shift))
	be.PutUint16(head[10:], uint16(n*16-pow*16))
	out.Write(head)
	offset := 12 + 16*n
	for _, t := range tables {
		rec := make([]byte, 16)
		copy(rec, t.tag[:])
		be.PutUint32(rec[4:], t.checksum)
		be.PutUint32(rec[8:], uint32(offset))
		be.PutUint32(rec[12:], uint32(len(t.data)))
		out.Write(rec)
		offset += (len(t.data) + 3) &^ 3
	}
	for _, t := range tables {
		out.Write(t.data)
		out.Write(make([]byte, ((len(t.data)+3)&^3)-len(t.data)))
	}
	return out.Bytes(), nil
}
