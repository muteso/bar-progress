package bar

import (
	"errors"
	"fmt"
)

const builderErr = "can't build Bar-struct:"

const (
	_ HorAlign = iota
	LEFT
	INSIDE
	RIGHT
)

const (
	_ UnitStyle = iota
	CURRENT
	CURRENT_AND_END
)

type (
	HorAlign  uint8
	UnitStyle uint8

	BarBuilder struct {
		open      string
		close     string
		seg       string
		empty_seg string
		right_tip string
		seg_len   int

		left_unit  int
		right_unit int
		direction  int8
		units      string

		hor_align  HorAlign
		unit_style UnitStyle

		before      func()
		after       func()
		on_overflow func(int)

		errs []error
	}

	Bar struct {
		unit_templ string
		bar_templ  string
		seg        string
		tip        string

		unit_templ_offset_left  int
		unit_templ_offset_right int
		unit_templ_max_len      int
		bar_start_offset        int
		bar_within_max_len      int

		seg_len int
		start   int
		end     int
		dir     int8

		before      func()
		after       func()
		on_overflow func(int)

		cur      int
		done     bool
		overflow int
	}
)

func NewBuilder() BarBuilder {
	return BarBuilder{
		open:       "[",
		close:      "]",
		seg:        "■",
		empty_seg:  " ",
		right_tip:  " ",
		seg_len:    5,
		left_unit:  0,
		right_unit: 100,
		direction:  1,
		units:      "%%",
		hor_align:  LEFT,
		unit_style: CURRENT,

		before:      func() {},
		after:       func() {},
		on_overflow: func(int) {},

		errs: make([]error, 0),
	}
}

func NewDefaultBar() Bar {
	return Bar{
		unit_templ:             "%d%%",
		bar_templ:              "[                    ]",
		seg:                    "■",
		tip:                    " ",
		unit_templ_offset_left: -1,
		unit_templ_max_len:     4,
		bar_start_offset:       1,
		bar_within_max_len:     20,
		seg_len:                5,
		start:                  0,
		end:                    100,
		dir:                    1,
		before:                 func() {},
		after:                  func() {},
		on_overflow:            func(int) {},
		cur:                    0,
		done:                   false,
		overflow:               0,
	}
}

func (bb BarBuilder) ConfUnits(left, right int) BarBuilder {
	if left == right {
		e := fmt.Errorf("can't set \"%d\" left border and \"%d\" right border of bar length: they are equal", left, right)
		bb.errs = append(bb.errs, e)
	} else {
		if left > right {
			bb.direction = -1
		} else {
			bb.direction = 1
		}
		bb.left_unit = left
		bb.right_unit = right
	}
	return bb
}

func (bb BarBuilder) ConfSegmentCost(segCost uint) BarBuilder {
	if int(segCost) > barLengthInUnits(bb.left_unit, bb.right_unit) {
		e := fmt.Errorf("can't set \"%d\" segment cost: it's more than overall bar lenght", segCost)
		bb.errs = append(bb.errs, e)
	} else {
		bb.seg_len = int(segCost)
	}
	return bb
}

func (bb BarBuilder) ConfBeforeAction(act func()) BarBuilder {
	bb.before = act
	return bb
}

func (bb BarBuilder) ConfAfterAction(act func()) BarBuilder {
	bb.after = act
	return bb
}

func (bb BarBuilder) ConfOnOverflowAction(act func(int)) BarBuilder {
	bb.on_overflow = act
	return bb
}

func (bb BarBuilder) ConfSegmentsRender(seg, emptySeg, endTip string) BarBuilder {
	if lenInRunes(seg) != 0 {
		bb.seg = protectString(seg)
	} else {
		e := fmt.Errorf("can't set \"%s\" segment for the bar: it has 0 length", seg)
		bb.errs = append(bb.errs, e)
	}
	if lenInRunes(bb.seg) == lenInRunes(emptySeg) {
		bb.empty_seg = protectString(emptySeg)
	} else {
		bb.empty_seg = " "
	}
	if lenInRunes(bb.seg) == lenInRunes(endTip) {
		bb.right_tip = protectString(endTip)
	} else {
		bb.right_tip = bb.empty_seg
	}
	return bb
}

func (bb BarBuilder) ConfBarRender(open, close string) BarBuilder {
	if lenInRunes(open) == lenInRunes(close) {
		bb.open = protectString(open)
		bb.close = protectString(close)
	} else {
		e := fmt.Errorf("can't set \"%s\" opening margin and \"%s\" closing margin of the bar: they don't have equal lenght", open, close)
		bb.errs = append(bb.errs, e)
	}
	return bb
}

func (bb BarBuilder) ConfUnitRender(units string, horAlign HorAlign, unitStyle UnitStyle) BarBuilder {
	bb.units = protectString(units)
	bb.hor_align = horAlign
	bb.unit_style = unitStyle
	return bb
}

func (bb BarBuilder) Build() (Bar, error) {
	if len(bb.errs) > 0 {
		e := builderErr
		for _, err := range bb.errs {
			e += fmt.Sprintf("\n%s", err.Error())
		}
		return Bar{}, errors.New(e)
	}
	if barLengthInUnits(bb.left_unit, bb.right_unit) < bb.seg_len {
		e := builderErr + fmt.Sprintf("\ncan't set \"%d\" segment length: it's more than overall bar lenght", bb.seg_len)
		return Bar{}, errors.New(e)
	}

	b := Bar{
		bar_start_offset: lenInRunes(bb.open),
		seg:              bb.seg,
		tip:              bb.right_tip,

		seg_len: bb.seg_len,
		start:   bb.left_unit,
		end:     bb.right_unit,
		dir:     bb.direction,

		before:      bb.before,
		after:       bb.after,
		on_overflow: bb.on_overflow,

		cur:  bb.left_unit,
		done: false,
	}
	b.buildTempls(bb)

	return b, nil
}

func (b *Bar) Start(progressChan <-chan int) error {
	if b.done {
		return errors.New("can't start progress bar print: current progress bar is already done - please reset it if you want to reuse it")
	}

	b.before()
	b.render(0)
	for !b.done {
		select {
		case p := <-progressChan:
			b.progress(p)
		default:
		}
	}
	fmt.Println()
	b.after()

	if b.overflow > 0 {
		b.on_overflow(b.overflow)
	}

	return nil
}

func (b *Bar) Stop() error {
	if b.done {
		return errors.New("can't stop progress bar print: current progress bar is already done - please reset it if you want to reuse it")
	}
	b.done = true
	return nil
}

func (b *Bar) Reset() {
	b.overflow = 0
	b.done = false
	b.cur = b.start
}
