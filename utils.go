package bar

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

func barLengthInUnits(left, right int) int {
	return int(math.Abs(float64(right - left)))
}

func lenInRunes(str string) int {
	return utf8.RuneCountInString(str)
}

func protectString(str string) string {
	switch {
	case strings.Contains(str, "%"):
		return strings.Replace(str, "%", "%%", -1)
	}
	return str
}

func (b *Bar) buildTempls(bb BarBuilder) {
	c := barLengthInUnits(bb.left_unit, bb.right_unit) / bb.seg_len
	if bb.right_unit%bb.seg_len != 0 {
		c++
	}
	b.bar_templ = bb.open + strings.Repeat(bb.empty_seg, c) + bb.close

	switch bb.unit_style {
	case CURRENT:
		b.unit_templ = "%d" + bb.units
	case CURRENT_AND_END:
		b.unit_templ = "%d/" + strconv.Itoa(b.end) + bb.units
	}

	switch bb.direction {
	case 1:
		b.unit_templ_max_len = lenInRunes(fmt.Sprintf(b.unit_templ, b.end))
	case -1:
		b.unit_templ_max_len = lenInRunes(fmt.Sprintf(b.unit_templ, b.start))
	}

	b.bar_within_max_len = lenInRunes(b.bar_templ) - lenInRunes(bb.open) - lenInRunes(bb.close)
	switch bb.hor_align {
	case LEFT:
		b.unit_templ_offset_left = -1
	case INSIDE:
		barCenter := b.bar_within_max_len / 2
		if (b.bar_within_max_len%2 != 0 && b.unit_templ_max_len%2 == 0) || (b.bar_within_max_len%2 == 0 && b.unit_templ_max_len%2 != 0) {
			b.unit_templ = strings.Replace(b.unit_templ, bb.units, " "+bb.units, 1)
			b.unit_templ_max_len++
		}
		unitTemplHalf := b.unit_templ_max_len / 2
		b.unit_templ_offset_left = barCenter - unitTemplHalf + b.bar_start_offset
		b.unit_templ_offset_right = barCenter + unitTemplHalf + b.bar_start_offset
		if b.unit_templ_max_len%2 != 0 {
			b.unit_templ_offset_right++
		}
	case RIGHT:
		b.unit_templ_offset_right = -1
	}
}

func (b *Bar) progress(prog int) {
	b.cur += prog * int(b.dir)

	if (b.dir == 1 && b.cur >= b.end) || (b.dir == -1 && b.cur <= b.end) {
		switch b.dir {
		case 1:
			b.overflow = b.cur - b.end
		case -1:
			b.overflow = b.end - b.cur
		}
		b.done = true
	}

	if count := (b.cur - b.start) * int(b.dir) / b.seg_len; count > 0 {
		if b.overflow > 0 {
			b.cur = b.end
			count -= b.overflow / b.seg_len
		}
		b.render(count)
	}
}

func (b Bar) render(segCount int) {
	bar := b.fillBarTempl(segCount)
	switch {
	case b.unit_templ_offset_left == -1:
		bar = b.unitTemplWithTrailingSpaces() + " " + bar + "\r"
	case b.unit_templ_offset_right == -1:
		bar = bar + " " + b.unitTemplWithTrailingSpaces() + "\r"
	default:
		bar = b.unitTemplWithinBar(bar) + "\r"
	}

	fmt.Printf(bar, b.cur)
}

func (b Bar) calcTrailingSpaces() int {
	return b.unit_templ_max_len - lenInRunes(fmt.Sprintf(b.unit_templ, b.cur))
}

func (b Bar) unitTemplWithTrailingSpaces() string {
	return strings.Repeat(" ", b.calcTrailingSpaces()) + b.unit_templ
}

func (b Bar) fillBarTempl(count int) string {
	var (
		bar   = strings.Repeat(string(b.seg), count)
		runeI int
		res   string
		left  = b.bar_start_offset
		right = b.bar_start_offset + lenInRunes(bar)
	)
	if lenInRunes(bar) != b.bar_within_max_len {
		bar += b.tip
		right++
	}
	for i := range b.bar_templ {
		if runeI == left {
			res = b.bar_templ[:i] + bar
		}
		if runeI == right {
			res = res + b.bar_templ[i:]
		}
		runeI++
	}
	return res
}

func (b Bar) unitTemplWithinBar(bar string) string {
	var (
		runeI int
		res   string
		left  = b.unit_templ_offset_left + b.calcTrailingSpaces()
		right = b.unit_templ_offset_right
	)
	for i := range bar {
		if runeI == left {
			res = bar[:i] + b.unit_templ
		}
		if runeI == right {
			res = res + bar[i:]
		}
		runeI++
	}
	return res
}
