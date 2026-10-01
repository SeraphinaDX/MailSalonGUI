// SPDX-License-Identifier: GPL-3.0-only
package gui

import (
	"image/color"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/pim"
)

const calendarHourHeight float32 = 64
const calendarTimeGutter float32 = 60

func calendarTint(alpha uint8) color.Color {
	c := color.NRGBAModel.Convert(theme.PrimaryColor()).(color.NRGBA)
	c.A = alpha
	return c
}

// A tile has a small stable minimum width, so long titles never force a month
// column or overlapping appointment wider than its available space.
type calendarTile struct {
	widget.BaseWidget
	title, detail string
	compact       bool
	tap           func()
	tooltip       *widget.PopUp
}

func newCalendarTile(title, detail string, compact bool, tap func()) *calendarTile {
	b := &calendarTile{title: title, detail: detail, compact: compact, tap: tap}
	b.ExtendBaseWidget(b)
	return b
}
func (b *calendarTile) Tapped(*fyne.PointEvent) {
	b.MouseOut()
	if b.tap != nil {
		b.tap()
	}
}
func (b *calendarTile) MouseIn(*desktop.MouseEvent) {
	if b.tap == nil {
		return
	}
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	cv := app.Driver().CanvasForObject(b)
	if cv == nil {
		return
	}
	text := b.title
	if b.detail != "" {
		text += "\n" + b.detail
	}
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	b.tooltip = widget.NewPopUp(label, cv)
	b.tooltip.Resize(fyne.NewSize(320, 64))
	b.tooltip.ShowAtRelativePosition(fyne.NewPos(0, b.Size().Height), b)
}
func (b *calendarTile) MouseMoved(*desktop.MouseEvent) {}
func (b *calendarTile) MouseOut() {
	if b.tooltip != nil {
		b.tooltip.Hide()
		b.tooltip = nil
	}
}
func (b *calendarTile) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(calendarTint(60))
	bg.CornerRadius = 4
	title := widget.NewLabel(b.title)
	title.Truncation = fyne.TextTruncateEllipsis
	detail := widget.NewLabel(b.detail)
	detail.Truncation = fyne.TextTruncateEllipsis
	return &calendarTileRenderer{tile: b, bg: bg, title: title, detail: detail}
}

type calendarTileRenderer struct {
	tile          *calendarTile
	bg            *canvas.Rectangle
	title, detail *widget.Label
}

func (r *calendarTileRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.title.Move(fyne.NewPos(1, 0))
	r.title.Resize(fyne.NewSize(size.Width-2, 24))
	if !r.tile.compact && size.Height >= 42 {
		r.detail.Show()
		r.detail.Move(fyne.NewPos(1, 22))
		r.detail.Resize(fyne.NewSize(size.Width-2, 24))
	} else {
		r.detail.Hide()
	}
}
func (r *calendarTileRenderer) MinSize() fyne.Size {
	if r.tile.compact {
		return fyne.NewSize(35, 24)
	}
	return fyne.NewSize(35, 48)
}
func (r *calendarTileRenderer) Refresh() {
	r.bg.FillColor = calendarTint(60)
	r.bg.Refresh()
	r.title.SetText(r.tile.title)
	r.detail.SetText(r.tile.detail)
}
func (r *calendarTileRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.title, r.detail}
}
func (r *calendarTileRenderer) Destroy() { r.tile.MouseOut() }

type calendarMonthCell struct {
	widget.BaseWidget
	day         time.Time
	inMonth     bool
	events      []pim.Occurrence
	openDay     func()
	selectEvent func(pim.Occurrence)
}

func newCalendarMonthCell(day time.Time, inMonth bool, events []pim.Occurrence, openDay func(), selectEvent func(pim.Occurrence)) *calendarMonthCell {
	c := &calendarMonthCell{day: day, inMonth: inMonth, events: events, openDay: openDay, selectEvent: selectEvent}
	c.ExtendBaseWidget(c)
	return c
}
func (c *calendarMonthCell) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.BackgroundColor())
	bg.StrokeColor = theme.DisabledColor()
	bg.StrokeWidth = 1
	date := newCalendarTile(c.day.Format("2"), "", true, c.openDay)
	if calendarMidnight(time.Now().In(c.day.Location())).Equal(c.day) {
		date.title += " · Today"
	}
	r := &calendarMonthRenderer{cell: c, bg: bg, date: date, more: newCalendarTile("", "", true, c.openDay)}
	for _, event := range c.events {
		o := event
		text := calendarEventText(o)
		if !o.AllDay && o.Start.Before(c.day) {
			text = "Continues · " + o.Title
		}
		r.events = append(r.events, newCalendarTile(text, "", true, func() { c.selectEvent(o) }))
	}
	return r
}

type calendarMonthRenderer struct {
	cell       *calendarMonthCell
	bg         *canvas.Rectangle
	date, more *calendarTile
	events     []*calendarTile
}

func (r *calendarMonthRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.date.Move(fyne.NewPos(3, 2))
	r.date.Resize(fyne.NewSize(size.Width-6, 24))
	slots := int((size.Height - 30) / 26)
	if slots < 0 {
		slots = 0
	}
	visible := slots
	overflow := len(r.events) > slots
	if overflow && visible > 0 {
		visible--
	}
	for i, tile := range r.events {
		if i < visible {
			tile.Show()
			tile.Move(fyne.NewPos(3, 28+float32(i)*26))
			tile.Resize(fyne.NewSize(size.Width-6, 24))
		} else {
			tile.Hide()
		}
	}
	if overflow && slots > 0 {
		r.more.title = calendarMoreLabel(len(r.events) - visible)
		r.more.Refresh()
		r.more.Show()
		r.more.Move(fyne.NewPos(3, 28+float32(visible)*26))
		r.more.Resize(fyne.NewSize(size.Width-6, 24))
	} else {
		r.more.Hide()
	}
}
func (r *calendarMonthRenderer) MinSize() fyne.Size { return fyne.NewSize(84, 86) }
func (r *calendarMonthRenderer) Refresh() {
	if r.cell.inMonth {
		r.bg.FillColor = theme.BackgroundColor()
	} else {
		r.bg.FillColor = theme.InputBackgroundColor()
	}
	r.bg.Refresh()
}
func (r *calendarMonthRenderer) Objects() []fyne.CanvasObject {
	out := []fyne.CanvasObject{r.bg, r.date}
	for _, tile := range r.events {
		out = append(out, tile)
	}
	return append(out, r.more)
}
func (r *calendarMonthRenderer) Destroy() {}

type calendarSegment struct {
	event            pim.Occurrence
	day, lane, lanes int
	top, bottom      float32
}

func calendarSegments(from time.Time, days int, events []pim.Occurrence) []calendarSegment {
	var result []calendarSegment
	for dayIndex := 0; dayIndex < days; dayIndex++ {
		day := from.AddDate(0, 0, dayIndex)
		end := day.AddDate(0, 0, 1)
		var segments []calendarSegment
		for _, o := range events {
			if o.AllDay || !occurrenceOnDay(o, day) {
				continue
			}
			start, stop := o.Start, o.End
			if start.Before(day) {
				start = day
			}
			if stop.After(end) {
				stop = end
			}
			minutes := func(t time.Time) float32 { return float32(t.Hour()*60+t.Minute()) + float32(t.Second())/60 }
			top, bottom := minutes(start)/60*calendarHourHeight, minutes(stop)/60*calendarHourHeight
			if stop.Equal(end) {
				bottom = 24 * calendarHourHeight
			}
			if bottom-top < 26 {
				bottom = top + 26
			}
			if bottom > 24*calendarHourHeight {
				bottom = 24 * calendarHourHeight
			}
			segments = append(segments, calendarSegment{event: o, day: dayIndex, top: top, bottom: bottom})
		}
		sort.SliceStable(segments, func(i, j int) bool {
			if segments[i].top == segments[j].top {
				return segments[i].bottom > segments[j].bottom
			}
			return segments[i].top < segments[j].top
		})
		// Connected overlap groups share a lane count; disjoint meetings reclaim
		// the full column width. This also separates visually tiny appointments.
		for groupStart := 0; groupStart < len(segments); {
			groupEnd := groupStart + 1
			endTime := segments[groupStart].bottom
			for groupEnd < len(segments) && segments[groupEnd].top < endTime {
				if segments[groupEnd].bottom > endTime {
					endTime = segments[groupEnd].bottom
				}
				groupEnd++
			}
			var laneEnds []float32
			for i := groupStart; i < groupEnd; i++ {
				lane := 0
				for lane < len(laneEnds) && laneEnds[lane] > segments[i].top {
					lane++
				}
				if lane == len(laneEnds) {
					laneEnds = append(laneEnds, segments[i].bottom)
				} else {
					laneEnds[lane] = segments[i].bottom
				}
				segments[i].lane = lane
			}
			for i := groupStart; i < groupEnd; i++ {
				segments[i].lanes = len(laneEnds)
			}
			groupStart = groupEnd
		}
		result = append(result, segments...)
	}
	return result
}

type calendarTimeGrid struct {
	widget.BaseWidget
	days        int
	segments    []calendarSegment
	selectEvent func(pim.Occurrence)
}

func newCalendarTimeGrid(from time.Time, days int, events []pim.Occurrence, selectEvent func(pim.Occurrence)) *calendarTimeGrid {
	g := &calendarTimeGrid{days: days, segments: calendarSegments(from, days, events), selectEvent: selectEvent}
	g.ExtendBaseWidget(g)
	return g
}
func (g *calendarTimeGrid) CreateRenderer() fyne.WidgetRenderer {
	r := &calendarTimeRenderer{grid: g, bg: canvas.NewRectangle(theme.BackgroundColor())}
	for hour := 0; hour <= 24; hour++ {
		r.lines = append(r.lines, canvas.NewLine(theme.DisabledColor()))
		if hour < 24 {
			label := canvas.NewText(calendarHourLabel(hour), theme.ForegroundColor())
			label.TextSize = 12
			r.labels = append(r.labels, label)
		}
	}
	for i := 0; i <= g.days; i++ {
		r.columns = append(r.columns, canvas.NewLine(theme.DisabledColor()))
	}
	for _, segment := range g.segments {
		o := segment.event
		tile := newCalendarTile(o.Title, o.Start.Format("15:04")+" – "+o.End.Format("15:04"), false, func() { g.selectEvent(o) })
		r.tiles = append(r.tiles, tile)
	}
	return r
}

type calendarTimeRenderer struct {
	grid           *calendarTimeGrid
	bg             *canvas.Rectangle
	lines, columns []*canvas.Line
	labels         []*canvas.Text
	tiles          []*calendarTile
}

func (r *calendarTimeRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	width := (size.Width - calendarTimeGutter) / float32(r.grid.days)
	for i, line := range r.lines {
		line.Position1 = fyne.NewPos(calendarTimeGutter, float32(i)*calendarHourHeight)
		line.Position2 = fyne.NewPos(size.Width, float32(i)*calendarHourHeight)
		line.Refresh()
	}
	for i, label := range r.labels {
		label.Move(fyne.NewPos(5, float32(i)*calendarHourHeight+3))
	}
	for i, line := range r.columns {
		x := calendarTimeGutter + float32(i)*width
		line.Position1 = fyne.NewPos(x, 0)
		line.Position2 = fyne.NewPos(x, size.Height)
		line.Refresh()
	}
	for i, segment := range r.grid.segments {
		tile := r.tiles[i]
		laneWidth := width / float32(segment.lanes)
		tile.Move(fyne.NewPos(calendarTimeGutter+float32(segment.day)*width+float32(segment.lane)*laneWidth+2, segment.top+1))
		tile.Resize(fyne.NewSize(laneWidth-4, segment.bottom-segment.top-2))
	}
}
func (r *calendarTimeRenderer) MinSize() fyne.Size { return fyne.NewSize(320, 24*calendarHourHeight) }
func (r *calendarTimeRenderer) Refresh()           { r.bg.FillColor = theme.BackgroundColor(); r.bg.Refresh() }
func (r *calendarTimeRenderer) Objects() []fyne.CanvasObject {
	out := []fyne.CanvasObject{r.bg}
	for _, line := range r.lines {
		out = append(out, line)
	}
	for _, line := range r.columns {
		out = append(out, line)
	}
	for _, label := range r.labels {
		out = append(out, label)
	}
	for _, tile := range r.tiles {
		out = append(out, tile)
	}
	return out
}
func (r *calendarTimeRenderer) Destroy() {}
