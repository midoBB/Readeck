// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package credentials

import (
	"bufio"
	cryptorand "crypto/rand"
	"embed"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"strings"
)

//go:embed assets/*.txt
var assets embed.FS

// MakePassphrase creates a passphrase based on the EFF short word list
// https://www.eff.org/deeplinks/2016/07/new-wordlists-random-passphrases
func MakePassphrase(size int) (string, error) {
	f, err := assets.Open("assets/wordlist.txt")
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck

	// Scan lines and store the word index
	wordIndex := [][2]int{}
	scanner := bufio.NewScanner(f)
	offset := 0

	for scanner.Scan() {
		wordIndex = append(wordIndex, [2]int{offset, len(scanner.Bytes())})
		offset += len(scanner.Bytes()) + 1
	}

	if size > len(wordIndex) {
		return "", fmt.Errorf("max passphrase words is %d", len(wordIndex))
	}

	s := f.(io.ReadSeeker)

	r := rand.New(&cryptorandSource{}) //nolint:gosec

	choice := r.Perm(len(wordIndex))
	res := make([]string, size)
	for i := range res {
		if _, err = s.Seek(int64(wordIndex[choice[i]][0]), io.SeekStart); err != nil {
			return "", err
		}
		b := make([]byte, wordIndex[choice[i]][1])
		if _, err = s.Read(b); err != nil {
			return "", err
		}
		res[i] = string(b)
	}

	return strings.Join(res, " "), nil
}

type cryptorandSource struct{}

func (c *cryptorandSource) Uint64() uint64 {
	var b [8]byte
	if _, err := io.ReadFull(cryptorand.Reader, b[:]); err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint64(b[:])
}
