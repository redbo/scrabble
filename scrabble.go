package main

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

// https://en.wikipedia.org/wiki/Scrabble_letter_distributions
var tilePoints = [255]int{'E': 1, 'A': 1, 'I': 1, 'O': 1, 'N': 1, 'R': 1, 'T': 1, 'L': 1, 'S': 1, 'U': 1, 'D': 2, 'G': 2, 'B': 3, 'C': 3, 'M': 3, 'P': 3, 'F': 4, 'H': 4, 'V': 4, 'W': 4, 'Y': 4, 'K': 5, 'J': 8, 'X': 8, 'Q': 10, 'Z': 10}
var startTiles = "AAAAAAAAABBCCDDDDEEEEEEEEEEEEFFGGGHHIIIIIIIIIJKLLLLMMNNNNNNOOOOOOOOPPQRRRRRRSSSSTTTTTTUUUUVVWWXYYZ"

var tw = [225]bool{0: true, 7: true, 14: true, 105: true, 210: true, 217: true, 224: true}
var dw = [225]bool{16: true, 28: true, 32: true, 42: true, 48: true, 56: true, 64: true, 70: true, 154: true, 160: true, 168: true, 176: true, 182: true, 192: true, 196: true, 208: true}
var tl = [225]bool{20: true, 24: true, 76: true, 80: true, 84: true, 88: true, 136: true, 140: true, 144: true, 148: true, 200: true, 204: true}
var dl = [225]bool{3: true, 11: true, 36: true, 38: true, 45: true, 52: true, 59: true, 92: true, 96: true, 98: true, 102: true, 108: true, 122: true, 126: true, 128: true, 132: true, 165: true, 172: true, 179: true, 186: true, 188: true, 213: true, 221: true}

type direction int

var DIR_VERT direction = 0
var DIR_HORIZ direction = 1

type Board struct {
	board    []byte
	tiles    []byte
	wordlist map[string]struct{}
	pscore   [2]int
	ptiles   [2][]byte
}

func cti(x int, y int) int {
	return (y * 15) + x
}

func NewBoard(dict string) *Board {
	board := &Board{}
	board.wordlist = make(map[string]struct{})
	f, err := os.Open(dict)
	if err != nil {
		fmt.Println("Unable to open dictionary", err)
		return nil
	}
	r := bufio.NewReader(f)
	for line, _, err := r.ReadLine(); err == nil; line, _, err = r.ReadLine() {
		board.AddWord(strings.TrimRight(string(line), "\r\n"))
	}
	board.board = make([]byte, 225)
	board.ptiles = [2][]byte{[]byte{}, []byte{}}
	board.tiles = []byte(startTiles)
	for i := range board.tiles {
		j := rand.Intn(i + 1)
		board.tiles[i], board.tiles[j] = board.tiles[j], board.tiles[i]
	}
	board.ptiles[0], board.tiles = board.tiles[:7], board.tiles[7:]
	board.ptiles[1], board.tiles = board.tiles[:7], board.tiles[7:]
	return board
}

func (b *Board) AddWord(word string) {
	if len(word) > 1 {
		b.wordlist[word] = struct{}{}
	}
}

func (b *Board) checkGeometry(x, y, tiles int, dir direction) bool {
	spaces := 0
	contiguous := false
	centerWasPlayed := b.board[cti(7, 7)] != 0
	centerPlayedHere := false
	if dir == DIR_VERT {
		for i := y; i < 15 && tiles > spaces; i++ {
			if b.board[cti(x, i)] == 0 {
				spaces++
			}
			if x > 0 && b.board[cti(x-1, i)] != 0 {
				contiguous = true
			} else if x < 14 && b.board[cti(x+1, i)] != 0 {
				contiguous = true
			} else if i > 0 && b.board[cti(x, i-1)] != 0 {
				contiguous = true
			} else if i < 14 && b.board[cti(x, i+1)] != 0 {
				contiguous = true
			}
		}
		if x == 7 && y <= 7 && (y+tiles) >= 7 {
			centerPlayedHere = true
		}
	} else {
		for i := x; i < 15 && tiles > spaces; i++ {
			if b.board[cti(i, y)] == 0 {
				spaces++
			}
			if i > 0 && b.board[cti(i-1, y)] != 0 {
				contiguous = true
			} else if i < 14 && b.board[cti(i+1, y)] != 0 {
				contiguous = true
			} else if y > 0 && b.board[cti(i, y-1)] != 0 {
				contiguous = true
			} else if y < 14 && b.board[cti(i, y+1)] != 0 {
				contiguous = true
			}
		}
		if y == 7 && x <= 7 && (x+tiles) >= 7 {
			centerPlayedHere = true
		}
	}
	return (centerPlayedHere || (centerWasPlayed && contiguous)) && spaces >= tiles
}

func (b *Board) evaluateMove(x, y int, tiles string, dir direction) (bool, int) {
	plays := make(map[int]byte)
	playPoints := 0

	if !b.checkGeometry(x, y, len(tiles), dir) {
		return false, 0
	}

	interped := func(x, y int) byte {
		i := cti(x, y)
		if val, ok := plays[i]; ok {
			return val
		}
		return b.board[i]
	}

	checkWord := func(x, y int, dir direction, primary bool) (bool, int) {
		points := 0
		doubleWord := 1
		tripleWord := 1
		fullword := []byte{}
		var x2, y2 int

		countLetter := func(x, y int) bool {
			idx := cti(x, y)
			char := b.board[idx]
			if val, ok := plays[idx]; ok {
				char = val
				if dw[idx] {
					doubleWord *= 2
				} else if tw[idx] {
					tripleWord *= 3
				} else if dl[idx] {
					points += tilePoints[char]
				} else if tl[idx] {
					points += tilePoints[char] + tilePoints[char]
				}
			}
			if char != 0 {
				fullword = append(fullword, char)
				points += tilePoints[char]
				return true
			}
			return false
		}

		if dir == DIR_VERT {
			for y2 = y; y2 > 0 && interped(x, y2-1) != 0; y2-- {
			}
			for ; y2 < 15 && countLetter(x, y2); y2++ {
			}
		} else {
			for x2 = x; x2 > 0 && interped(x2-1, y) != 0; x2-- {
			}
			for ; x2 < 15 && countLetter(x2, y); x2++ {
			}
		}
		if len(fullword) == 1 {
			if primary {
				return false, 0
			} else {
				return true, 0
			}
		} else if _, ok := b.wordlist[string(fullword)]; !ok {
			return false, 0
		}
		return true, points * tripleWord * doubleWord
	}

	if dir == DIR_VERT {
		for i := y; len(tiles) > 0; i++ {
			if b.board[cti(x, i)] == 0 {
				plays[cti(x, i)] = tiles[0]
				tiles = tiles[1:]
				if valid, points := checkWord(x, i, DIR_HORIZ, false); valid {
					playPoints += points
				} else {
					return false, 0
				}
			}
		}

		if valid, points := checkWord(x, y, DIR_VERT, true); valid {
			playPoints += points
		} else {
			return false, 0
		}
	} else {
		for i := x; len(tiles) > 0; i++ {
			if b.board[cti(i, y)] == 0 {
				plays[cti(i, y)] = tiles[0]
				tiles = tiles[1:]
				if valid, points := checkWord(i, y, DIR_VERT, false); valid {
					playPoints += points
				} else {
					return false, 0
				}
			}
		}
		if valid, points := checkWord(x, y, DIR_HORIZ, true); valid {
			playPoints += points
		} else {
			return false, 0
		}
	}

	if interped(7, 7) == 0 {
		return false, 0
	}

	return true, playPoints
}

func (b *Board) play(x, y int, word string, dir direction) {
	if dir == DIR_VERT {
		for i := y; len(word) > 0; i++ {
			if b.board[cti(x, i)] != 0 {
				continue
			}
			b.board[cti(x, i)] = word[0]
			word = word[1:]
		}
	} else {
		for i := x; len(word) > 0; i++ {
			if b.board[cti(i, y)] != 0 {
				continue
			}
			b.board[cti(i, y)] = word[0]
			word = word[1:]
		}
	}
}

func (b *Board) PrintBoard() {
	for y := 0; y < 15; y++ {
		line := ""
		for x := 0; x < 15; x++ {
			if b.board[cti(x, y)] == 0 {
				line += "-"
			} else {
				line += string(b.board[cti(x, y)])
			}
			line += " "
		}
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println("Player 1:", b.pscore[0])
	fmt.Println("Player 2:", b.pscore[1])
}

func permute(s []byte) []string {
	var _permute func(s []byte, d int, result []string) []string
	_permute = func(s []byte, d int, result []string) []string {
		if d == len(s) {
			result = append(result, string(s))
		} else {
			for i := d; i < len(s); i++ {
				s[d], s[i] = s[i], s[d]
				result = _permute(s, d+1, result)
				s[d], s[i] = s[i], s[d]
			}
		}
		return result
	}
	subsets := map[string]bool{}
	for _, perm := range _permute(s, 0, nil) {
		for i := 1; i <= len(s); i++ {
			subsets[perm[:i]] = true
		}
	}
	keys := make([]string, 0, len(subsets))
	for key, _ := range subsets {
		if len(key) > 0 {
			keys = append(keys, key)
		}
	}
	return keys
}

func (b *Board) DoTurn(player int) {
	var playX, playY, playPoints int
	var playWord string
	var playDir direction
	plays := permute(b.ptiles[player])
	for x := 0; x < 15; x++ {
		for y := 0; y < 15; y++ {
			if b.board[cti(x, y)] != 0 {
				continue
			}
			for _, word := range plays {
				for _, dir := range []direction{DIR_HORIZ, DIR_VERT} {
					if validPlay, points := b.evaluateMove(x, y, word, dir); validPlay && points > playPoints {
						playX = x
						playY = y
						// fmt.Println("Switching to", word, x, y, dir)
						playWord = word
						playPoints = points
						playDir = dir
					}
				}
			}
		}
	}
	if playWord == "" {
		fmt.Println("NO WORD FOUND - PASSING")
		return
	}
	b.play(playX, playY, playWord, playDir)
	for _, c := range playWord {
		idx := bytes.IndexRune(b.ptiles[player], c)
		b.ptiles[player] = append(b.ptiles[player][:idx], b.ptiles[player][idx+1:]...)
	}
	for len(b.ptiles[player]) < 7 && len(b.tiles) > 0 {
		b.ptiles[player] = append(b.ptiles[player], b.tiles[0])
		b.tiles = b.tiles[1:]
	}
	b.pscore[player] += playPoints
}

func (b *Board) PlayersHaveTiles() bool {
	return len(b.ptiles[0]) > 0 && len(b.ptiles[1]) > 0
}

func main() {
	rand.Seed(time.Now().Unix())

	b := NewBoard("dictionary.txt")
	for b.PlayersHaveTiles() {
		b.DoTurn(0)
		b.DoTurn(1)
		b.PrintBoard()
	}
}
