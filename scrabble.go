package main

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"time"
)

type FNV struct {
	v uint64
	// debug []byte
}

func NewFNV() FNV {
	return FNV{v: 0xcbf29ce484222325}
}

func (h *FNV) Add(b byte) {
	h.v *= 0x100000001b3
	h.v ^= uint64(b & ^byte(32))
	// h.debug = append(h.debug, b)
}

func (h *FNV) AddString(s string) {
	for _, c := range s {
		h.v *= 0x100000001b3
		h.v ^= uint64(c)
	}
}

func (h *FNV) Val() uint64 {
	return uint64(h.v)
}

// https://en.wikipedia.org/wiki/Scrabble_letter_distributions
var tilePoints = [255]int{'E': 1, 'A': 1, 'I': 1, 'O': 1, 'N': 1, 'R': 1, 'T': 1, 'L': 1, 'S': 1, 'U': 1, 'D': 2, 'G': 2, 'B': 3, 'C': 3, 'M': 3, 'P': 3, 'F': 4, 'H': 4, 'V': 4, 'W': 4, 'Y': 4, 'K': 5, 'J': 8, 'X': 8, 'Q': 10, 'Z': 10}
var startTiles = "AAAAAAAAABBCCDDDDEEEEEEEEEEEEFFGGGHHIIIIIIIIIJKLLLLMMNNNNNNOOOOOOOOPPQRRRRRRSSSSTTTTTTUUUUVVWWXYYZ**"

var tw = [225]bool{0: true, 7: true, 14: true, 105: true, 210: true, 217: true, 224: true}
var dw = [225]bool{16: true, 28: true, 32: true, 42: true, 48: true, 56: true, 64: true, 70: true, 112: true, 154: true, 160: true, 168: true, 176: true, 182: true, 192: true, 196: true, 208: true}
var tl = [225]bool{20: true, 24: true, 76: true, 80: true, 84: true, 88: true, 136: true, 140: true, 144: true, 148: true, 200: true, 204: true}
var dl = [225]bool{3: true, 11: true, 36: true, 38: true, 45: true, 52: true, 59: true, 92: true, 96: true, 98: true, 102: true, 108: true, 122: true, 126: true, 128: true, 132: true, 165: true, 172: true, 179: true, 186: true, 188: true, 213: true, 221: true}

type direction int

var DIR_VERT direction = 0
var DIR_HORIZ direction = 1

type Board struct {
	board    [][]byte
	tiles    []byte
	wordlist map[uint64]struct{}
	pscore   [2]int
	ptiles   [2][]byte
}

func cti(x int, y int) int {
	return (y * 15) + x
}

func NewBoard(dict string) *Board {
	board := &Board{}
	board.wordlist = make(map[uint64]struct{})
	board.board = make([][]byte, 15)
	for i := 0; i < 15; i++ {
		board.board[i] = make([]byte, 15)
	}
	board.ptiles = [2][]byte{[]byte{}, []byte{}}
	board.tiles = []byte(startTiles)
	for i := range board.tiles {
		j := rand.Intn(i + 1)
		board.tiles[i], board.tiles[j] = board.tiles[j], board.tiles[i]
	}
	f, err := os.Open(dict)
	if err != nil {
		fmt.Println("Unable to open dictionary", err)
		return nil
	}
	r := bufio.NewReader(f)
	for line, _, err := r.ReadLine(); err == nil; line, _, err = r.ReadLine() {
		word := strings.TrimRight(string(line), "\r\n")
		if len(word) > 1 {
			board.addWord(word)
		}
	}
	board.ptiles[0], board.tiles = board.tiles[:7], board.tiles[7:]
	board.ptiles[1], board.tiles = board.tiles[:7], board.tiles[7:]
	return board
}

func (b *Board) addWord(word string) {
	f := NewFNV()
	f.AddString(word)
	b.wordlist[f.Val()] = struct{}{}
}
func (b *Board) checkCenterPlayed(x, y, tiles int, dir direction) bool {
	if b.board[7][7] != 0 {
		return true
	}
	if dir == DIR_VERT {
		return x == 7 && y <= 7 && (y+tiles) >= 7
	} else {
		return y == 7 && x <= 7 && (x+tiles) >= 7
	}
}

func (b *Board) checkContiguous(x, y, tiles int, dir direction) bool {
	if b.board[7][7] == 0 {
		return true
	}
	if dir == DIR_VERT {
		for i := y; tiles > 0; i++ {
			if b.board[x][i] == 0 {
				tiles--
			}
			if (x > 0 && b.board[x-1][i] != 0) || (x < 14 && b.board[x+1][i] != 0) || (i > 0 && b.board[x][i-1] != 0) || (i < 14 && b.board[x][i+1] != 0) {
				return true
			}
		}
	} else {
		for i := x; tiles > 0; i++ {
			if b.board[i][y] == 0 {
				tiles--
			}
			if (i > 0 && b.board[i-1][y] != 0) || (i < 14 && b.board[i+1][y] != 0) || (y > 0 && b.board[i][y-1] != 0) || (y < 14 && b.board[i][y+1] != 0) {
				return true
			}
		}
	}
	return false
}

func (b *Board) scoreWord(x, y int, dir direction, plays []byte) int {
	points := 0
	wordMult := 1
	var x2, y2 int
	wordLen := 0

	if dir == DIR_VERT {
		for y2 = y; y2 > 0 && (plays[cti(x, y2-1)] != 0 || b.board[x][y2-1] != 0); y2-- {
		}
		for ; y2 < 15; y2++ {
			idx := cti(x, y2)
			if b.board[x][y2] != 0 {
				wordLen++
				points += tilePoints[b.board[x][y2]]
			} else if plays[idx] != 0 {
				wordLen++
				points += tilePoints[plays[idx]]
				if dw[idx] {
					wordMult *= 2
				} else if tw[idx] {
					wordMult *= 3
				} else if dl[idx] {
					points += tilePoints[plays[idx]]
				} else if tl[idx] {
					points += tilePoints[plays[idx]] + tilePoints[plays[idx]]
				}
			} else {
				break
			}
		}
	} else {
		for x2 = x; x2 > 0 && (plays[cti(x2-1, y)] != 0 || b.board[x2-1][y] != 0); x2-- {
		}
		for ; x2 < 15; x2++ {
			idx := cti(x2, y)
			if b.board[x2][y] != 0 {
				wordLen++
				points += tilePoints[b.board[x2][y]]
			} else if plays[idx] != 0 {
				wordLen++
				points += tilePoints[plays[idx]]
				if dw[idx] {
					wordMult *= 2
				} else if tw[idx] {
					wordMult *= 3
				} else if dl[idx] {
					points += tilePoints[plays[idx]]
				} else if tl[idx] {
					points += tilePoints[plays[idx]] + tilePoints[plays[idx]]
				}
			} else {
				break
			}
		}
	}
	if wordLen == 1 {
		return 0
	}
	return points * wordMult
}

func (b *Board) scoreMove(x, y int, tiles string, dir direction) int {
	playPoints := 0
	tilei := 0
	plays := make([]byte, 225)

	if dir == DIR_VERT {
		for i := y; len(tiles) > tilei; i++ {
			if b.board[x][i] == 0 {
				plays[cti(x, i)] = tiles[tilei]
				tilei++
				playPoints += b.scoreWord(x, i, DIR_HORIZ, plays)
			}
		}
	} else {
		for i := x; len(tiles) > tilei; i++ {
			if b.board[i][y] == 0 {
				plays[cti(i, y)] = tiles[tilei]
				tilei++
				playPoints += b.scoreWord(i, y, DIR_VERT, plays)
			}
		}
	}
	return playPoints + b.scoreWord(x, y, dir, plays)
}

func (b *Board) getPlaySpace(x, y int, dir direction) ([]byte, [][]byte, int) {
	play := make([]byte, 0)
	crossPlays := make([][]byte, 0)
	spaces := 0
	if dir == DIR_VERT {
		for y = y; y > 0 && b.board[x][y-1] != 0; y-- {
		}
		for i := y; i < 15; i++ {
			play = append(play, b.board[x][i])
			var crossPlay []byte = nil
			if b.board[x][i] == 0 {
				spaces++
				var x2, x3 int
				for x2 = x; x2 > 0 && b.board[x2-1][i] != 0; x2-- {
				}
				for x3 = x; x3 < 14 && b.board[x3+1][i] != 0; x3++ {
				}
				if x2 < x3 {
					for j := x2; j <= x3; j++ {
						crossPlay = append(crossPlay, b.board[j][i])
					}
				}
			}
			crossPlays = append(crossPlays, crossPlay)
		}
	} else {
		for x = x; x > 0 && b.board[x-1][y] != 0; x-- {
		}
		for i := x; i < 15; i++ {
			play = append(play, b.board[i][y])
			var crossPlay []byte = nil
			if b.board[i][y] == 0 {
				spaces++
				var y2, y3 int
				for y2 = y; y2 > 0 && b.board[i][y2-1] != 0; y2-- {
				}
				for y3 = y; y3 < 14 && b.board[i][y3+1] != 0; y3++ {
				}
				if y2 < y3 {
					for j := y2; j <= y3; j++ {
						crossPlay = append(crossPlay, b.board[i][j])
					}
				}
			}
			crossPlays = append(crossPlays, crossPlay)
		}
	}
	return play, crossPlays, spaces
}

func (b *Board) play(x, y int, word string, dir direction) {
	if dir == DIR_VERT {
		for i := y; len(word) > 0; i++ {
			if b.board[x][i] != 0 {
				continue
			}
			b.board[x][i] = word[0]
			word = word[1:]
		}
	} else {
		for i := x; len(word) > 0; i++ {
			if b.board[i][y] != 0 {
				continue
			}
			b.board[i][y] = word[0]
			word = word[1:]
		}
	}
}

func (b *Board) PrintBoard() {
	for y := 0; y < 15; y++ {
		line := ""
		for x := 0; x < 15; x++ {
			if b.board[x][y] == 0 {
				if dw[cti(x, y)] {
					line += "\x1b[31;1m"
				} else if tw[cti(x, y)] {
					line += "\x1b[33;1m"
				} else if dl[cti(x, y)] {
					line += "\x1b[34;1m"
				} else if tl[cti(x, y)] {
					line += "\x1b[32;1m"
				}
				line += "."
			} else {
				line += string(b.board[x][y])
			}
			line += "\x1b[0m "
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
	subsets := make(map[string]struct{})
	for _, perm := range _permute(s, 0, nil) {
		for i := 1; i <= len(s); i++ {
			subsets[perm[:i]] = struct{}{}
		}
	}
	keys := make([]string, 0, len(subsets))
	for key := range subsets {
		if len(key) > 0 {
			switch strings.Count(key, "*") {
			case 1:
				wi := strings.Index(key, "*")
				for c := 'a'; c <= 'z'; c++ {
					keys = append(keys, key[:wi]+string(c)+key[wi+1:])
				}
			case 0:
				keys = append(keys, key)
			}
		}
	}
	for i := len(keys) - 1; i > 0; i-- {
		j := rand.Int() % i
		keys[i], keys[j] = keys[j], keys[i]
	}
	return keys
}

func (b *Board) DoTurn(player int) {
	var playX, playY, playPoints int
	var playTiles string
	var playDir direction
	tilesets := permute(b.ptiles[player])

	for _, x := range rand.Perm(15) {
		for _, y := range rand.Perm(15) {
			if b.board[x][y] != 0 {
				continue
			}
			for _, dir := range []direction{DIR_HORIZ, DIR_VERT} {
				play, crossPlays, room := b.getPlaySpace(x, y, dir)
				// fmt.Println("getPlaySpace", x, y, dir, ":", play, crossPlays, room)
				if room > len(b.ptiles[player]) {
					room = len(b.ptiles[player])
				}
				for tilecount := 1; tilecount <= room; tilecount++ {
					if !b.checkCenterPlayed(x, y, tilecount, dir) || !b.checkContiguous(x, y, tilecount, dir) {
						continue
					}
				TILESETLIST:
					for _, tileset := range tilesets {
						if len(tileset) != tilecount {
							continue
						}
						f := NewFNV()
						j := 0
						for i, v := range play {
							if v == 0 {
								if j == len(tileset) {
									break
								}
								f.Add(tileset[j])
								if crossPlays[i] != nil {
									f2 := NewFNV()
									for _, v := range crossPlays[i] {
										if v == 0 {
											f2.Add(tileset[j])
										} else {
											f2.Add(v)
										}
									}
									if _, ok := b.wordlist[f2.Val()]; !ok {
										continue TILESETLIST
									}
								}
								j++
							} else {
								f.Add(v)
							}
						}
						if _, ok := b.wordlist[f.Val()]; !ok {
							continue TILESETLIST
						}
						if points := b.scoreMove(x, y, tileset, dir); points > playPoints {
							playX = x
							playY = y
							playTiles = tileset
							playPoints = points
							playDir = dir
						}
					}
				}
			}
		}
	}
	if playTiles == "" {
		fmt.Println("NO WORD FOUND - PASSING")
		return
	}
	b.play(playX, playY, playTiles, playDir)
	fmt.Println("Play", playTiles, "for", playPoints, "points")
	for _, c := range playTiles {
		if c >= 'a' && c <= 'z' {
			c = '*'
		}
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
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().Unix())

	b := NewBoard("dictionary.txt")
	for b.PlayersHaveTiles() {
		b.DoTurn(0)
		b.DoTurn(1)
	}
	b.PrintBoard()
}
