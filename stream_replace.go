package main

import (
	"bytes"
	"io"
)

const streamBufferSize = 256 * 1024

// streamReplace copies from r to w, replacing every occurrence of find with replace.
// It returns whether any replacement was made. Memory use is bounded by streamBufferSize
// plus len(find) bytes of carry-over between reads.
func streamReplace(r io.Reader, w io.Writer, find, replace []byte) (bool, error) {
	if len(find) == 0 {
		return false, nil
	}

	buf := make([]byte, streamBufferSize)
	var pending []byte
	var changed bool

	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			data := append(pending, buf[:n]...)
			isFinal := readErr == io.EOF
			if !isFinal && len(data) < streamBufferSize {
				pending = data
				continue
			}
			out, rest, chunkChanged := replaceChunk(data, find, replace, isFinal)
			if chunkChanged {
				changed = true
			}
			if len(out) > 0 {
				if _, err := w.Write(out); err != nil {
					return changed, err
				}
			}
			pending = rest
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return changed, readErr
		}
	}

	if len(pending) > 0 {
		out, _, chunkChanged := replaceChunk(pending, find, replace, true)
		if chunkChanged {
			changed = true
		}
		if len(out) > 0 {
			if _, err := w.Write(out); err != nil {
				return changed, err
			}
		}
	}

	return changed, nil
}

func replaceChunk(data, find, replace []byte, final bool) (out []byte, rest []byte, changed bool) {
	if len(data) == 0 {
		return nil, nil, false
	}

	if final {
		replaced := bytes.Replace(data, find, replace, -1)
		return replaced, nil, !bytes.Equal(replaced, data)
	}

	overlap := len(find) - 1
	if overlap >= len(data) {
		return nil, append([]byte(nil), data...), false
	}

	split := len(data) - overlap
	process := data[:split]
	rest = append([]byte(nil), data[split:]...)

	replaced := bytes.Replace(process, find, replace, -1)
	return replaced, rest, !bytes.Equal(replaced, process)
}
