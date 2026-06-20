package spark

import (
	"math"
	"strings"
)

func renderArt(source string, maxWidth, maxHeight int) string {
	lines := cropLines(strings.Split(strings.Trim(source, "\n"), "\n"))
	if len(lines) == 0 || maxWidth <= 0 || maxHeight <= 0 {
		return ""
	}

	sourceHeight := len(lines)
	sourceWidth := 0
	for _, line := range lines {
		if width := len([]rune(line)); width > sourceWidth {
			sourceWidth = width
		}
	}
	if sourceWidth == 0 {
		return ""
	}

	targetWidth := sourceWidth
	targetHeight := sourceHeight
	scale := math.Max(float64(sourceWidth)/float64(maxWidth), float64(sourceHeight)/float64(maxHeight))
	if scale > 1 {
		targetWidth = max(1, int(math.Ceil(float64(sourceWidth)/scale)))
		targetHeight = max(1, int(math.Ceil(float64(sourceHeight)/scale)))
	}

	grid := make([][]rune, sourceHeight)
	for y, line := range lines {
		row := []rune(line)
		if len(row) < sourceWidth {
			row = append(row, []rune(strings.Repeat(" ", sourceWidth-len(row)))...)
		}
		grid[y] = row
	}

	out := make([]string, 0, targetHeight)
	for y := 0; y < targetHeight; y++ {
		y0 := int(math.Floor(float64(y) * float64(sourceHeight) / float64(targetHeight)))
		y1 := int(math.Ceil(float64(y+1) * float64(sourceHeight) / float64(targetHeight)))
		if y1 <= y0 {
			y1 = y0 + 1
		}

		var b strings.Builder
		for x := 0; x < targetWidth; x++ {
			x0 := int(math.Floor(float64(x) * float64(sourceWidth) / float64(targetWidth)))
			x1 := int(math.Ceil(float64(x+1) * float64(sourceWidth) / float64(targetWidth)))
			if x1 <= x0 {
				x1 = x0 + 1
			}
			b.WriteRune(densestRune(grid, x0, x1, y0, y1))
		}
		out = append(out, strings.TrimRight(b.String(), " "))
	}

	return strings.Trim(strings.Join(out, "\n"), "\n")
}

func renderBraille(source string) string {
	lines := cropLines(strings.Split(strings.Trim(source, "\n"), "\n"))
	if len(lines) == 0 {
		return ""
	}

	height := len(lines)
	width := 0
	for _, line := range lines {
		if w := len([]rune(line)); w > width {
			width = w
		}
	}
	if width == 0 {
		return ""
	}

	grid := make([][]rune, height)
	for y, line := range lines {
		row := []rune(line)
		if len(row) < width {
			row = append(row, []rune(strings.Repeat(" ", width-len(row)))...)
		}
		grid[y] = row
	}

	var out []string
	for y := 0; y < height; y += 4 {
		var b strings.Builder
		for x := 0; x < width; x += 2 {
			mask := 0
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					yy := y + dy
					xx := x + dx
					if yy >= height || xx >= width || grid[yy][xx] == ' ' {
						continue
					}
					mask |= brailleDot(dx, dy)
				}
			}
			b.WriteRune(rune(0x2800 + mask))
		}
		out = append(out, strings.TrimRight(b.String(), "⠀"))
	}
	return strings.Trim(strings.Join(out, "\n"), "\n")
}

func brailleDot(x, y int) int {
	switch {
	case x == 0 && y == 0:
		return 0x01
	case x == 0 && y == 1:
		return 0x02
	case x == 0 && y == 2:
		return 0x04
	case x == 0 && y == 3:
		return 0x40
	case x == 1 && y == 0:
		return 0x08
	case x == 1 && y == 1:
		return 0x10
	case x == 1 && y == 2:
		return 0x20
	case x == 1 && y == 3:
		return 0x80
	default:
		return 0
	}
}

func cropLines(lines []string) []string {
	top := 0
	for top < len(lines) && strings.TrimSpace(lines[top]) == "" {
		top++
	}
	bottom := len(lines) - 1
	for bottom >= top && strings.TrimSpace(lines[bottom]) == "" {
		bottom--
	}
	if top > bottom {
		return nil
	}

	left := -1
	right := 0
	for _, line := range lines[top : bottom+1] {
		runes := []rune(line)
		for i, r := range runes {
			if r == ' ' {
				continue
			}
			if left == -1 || i < left {
				left = i
			}
			if i+1 > right {
				right = i + 1
			}
		}
	}
	if left == -1 {
		return nil
	}

	cropped := make([]string, 0, bottom-top+1)
	for _, line := range lines[top : bottom+1] {
		runes := []rune(line)
		if len(runes) < right {
			runes = append(runes, []rune(strings.Repeat(" ", right-len(runes)))...)
		}
		cropped = append(cropped, string(runes[left:right]))
	}
	return cropped
}

func densestRune(grid [][]rune, x0, x1, y0, y1 int) rune {
	best := ' '
	bestRank := 0
	total := 0
	filled := 0
	for y := y0; y < y1 && y < len(grid); y++ {
		for x := x0; x < x1 && x < len(grid[y]); x++ {
			total++
			r := grid[y][x]
			if r != ' ' {
				filled++
			}
			if rank := pixelRank(r); rank > bestRank {
				best = r
				bestRank = rank
			}
		}
	}
	if total > 1 && float64(filled)/float64(total) < 0.35 {
		return ' '
	}
	return best
}

func pixelRank(r rune) int {
	switch r {
	case '█':
		return 5
	case '▓':
		return 4
	case '▒':
		return 3
	case '░':
		return 2
	case ' ':
		return 0
	default:
		return 1
	}
}
