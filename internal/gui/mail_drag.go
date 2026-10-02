// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fmt"
	"image/color"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/SeraphinaDX/MailSalonGUI/internal/maildir"
)

// A drag captures view/selection identity, rather than file paths that an
// automatic mark-read rename may change before the user releases the pointer.
type mailDrag struct {
	account              int
	generation, revision uint64
	source               string
	position             fyne.Position
	count                int
	target               *folderRow
}

func (a *App) startMailDrag(id int) bool {
	if id < 0 || id >= len(a.messages) || a.tabs.SelectedIndex() != 0 || a.Window.Canvas().Overlays().Top() != nil || (a.changing && !a.changingRead) || a.syncing || a.folderBusy {
		return false
	}
	a.cancelMailDrag()
	a.hideMessageMenu()
	a.Window.Canvas().Focus(a.messageList)
	if !a.messageList.selection[id] {
		a.messageList.Select(id)
	}
	a.mailDrag = &mailDrag{account: a.account, generation: a.generation, revision: a.messageList.revision, source: a.folderPath, count: len(a.messageList.SelectedIDs())}
	return a.mailDrag.count > 0
}

func (a *App) validMailDrag(d *mailDrag) bool {
	return d != nil && a.ctx.Err() == nil && a.account == d.account && a.generation == d.generation && a.messageList.revision == d.revision && a.folderPath == d.source && a.tabs.SelectedIndex() == 0 && a.Window.Canvas().Overlays().Top() == nil && !a.syncing && !a.folderBusy && (!a.changing || a.changingRead)
}
func insideObject(a *App, object fyne.CanvasObject, point fyne.Position) bool {
	if !object.Visible() {
		return false
	}
	pos := a.Fyne.Driver().AbsolutePositionForObject(object)
	size := object.Size()
	return point.X >= pos.X && point.Y >= pos.Y && point.X < pos.X+size.Width && point.Y < pos.Y+size.Height
}
func (a *App) dragDestination(d *mailDrag) *folderRow {
	r := a.folderHover
	if r == nil || r.id < 0 || r.id >= len(a.folders) || r.path != a.folderHoverPath || a.folders[r.id].Path != r.path || filepath.Clean(r.path) == filepath.Clean(d.source) {
		return nil
	}
	if !insideObject(a, a.folderList, d.position) || !insideObject(a, r, d.position) {
		return nil
	}
	return r
}
func (a *App) updateMailDrag(position fyne.Position) {
	d := a.mailDrag
	if !a.validMailDrag(d) {
		a.cancelMailDrag()
		return
	}
	d.position = position
	if d.target != nil {
		d.target.setDrop(false)
	}
	d.target = a.dragDestination(d)
	noun := "messages"
	if d.count == 1 {
		noun = "message"
	}
	if d.target != nil {
		d.target.setDrop(true)
		a.status.SetText(fmt.Sprintf("Move %d %s to %s — release to move", d.count, noun, a.folders[d.target.id].Name))
	} else {
		a.status.SetText(fmt.Sprintf("Moving %d %s — drop on a folder; Esc cancels", d.count, noun))
	}
}
func (a *App) hoverFolder(row *folderRow) {
	a.folderHover = row
	a.folderHoverPath = ""
	if row != nil {
		a.folderHoverPath = row.path
	}
	if a.mailDrag != nil {
		a.updateMailDrag(a.mailDrag.position)
	}
}
func (a *App) clearMailDrag() *mailDrag {
	d := a.mailDrag
	a.mailDrag = nil
	if d != nil && d.target != nil {
		d.target.setDrop(false)
	}
	return d
}
func (a *App) cancelMailDrag() {
	if a.clearMailDrag() != nil {
		a.status.SetText("Move cancelled")
	}
}
func (a *App) finishMailDrag() {
	d := a.clearMailDrag()
	if !a.validMailDrag(d) {
		return
	}
	r := a.dragDestination(d)
	if r == nil {
		a.status.SetText("Move cancelled — drop on a different folder")
		return
	}
	destination := a.folders[r.id]
	// Recheck after a pending mark-read rename, using the current selected paths.
	a.withSelection(func(entries []maildir.Entry) {
		if !a.validMailDrag(d) {
			return
		}
		a.mutateMessages(entries, "Moved to "+destination.Name, true, func(e *maildir.Entry) error { return maildir.Move(*e, destination) })
	})
}

// Folder hover remains independent of folder selection. Dropping never opens
// the destination or changes the source view before the move finishes.
type folderRow struct {
	widget.BaseWidget
	app        *App
	label      *widget.Label
	background *canvas.Rectangle
	id         int
	path       string
	drop       bool
}

func newFolderRow(a *App) *folderRow {
	r := &folderRow{app: a, id: -1, label: widget.NewLabel("")}
	r.label.Truncation = fyne.TextTruncateEllipsis
	r.background = canvas.NewRectangle(color.Transparent)
	r.ExtendBaseWidget(r)
	return r
}
func (r *folderRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(r.background, r.label))
}
func (r *folderRow) setDrop(drop bool) {
	if r.drop == drop {
		return
	}
	r.drop = drop
	r.background.FillColor = color.Transparent
	r.background.StrokeWidth = 0
	if drop {
		r.background.FillColor = theme.Color(theme.ColorNameHover)
		r.background.StrokeColor = theme.Color(theme.ColorNamePrimary)
		r.background.StrokeWidth = 2
	}
	r.background.Refresh()
}
func (r *folderRow) MouseIn(*desktop.MouseEvent)    { r.app.hoverFolder(r) }
func (r *folderRow) MouseMoved(*desktop.MouseEvent) { r.app.hoverFolder(r) }
func (r *folderRow) MouseOut() {
	if r.app.folderHover == r {
		r.app.hoverFolder(nil)
	}
}

// Fyne's desktop hit testing selects this row for both mouse buttons because
// it implements SecondaryTappable. Forward primary taps to the list so its
// selection highlight and OnSelected callback still run.
func (r *folderRow) Tapped(*fyne.PointEvent) {
	if r.app.mailDrag != nil || r.id < 0 || r.id >= len(r.app.folders) || r.path != r.app.folders[r.id].Path {
		return
	}
	r.app.folderList.Select(r.id)
}

func (r *folderRow) TappedSecondary(event *fyne.PointEvent) {
	menu := fyne.NewMenu("Folders", fyne.NewMenuItem("New folder…", func() { r.app.showFolderManager(true) }), fyne.NewMenuItem("Manage subscriptions…", func() { r.app.showFolderManager(false) }))
	widget.ShowPopUpMenuAtPosition(menu, r.app.Window.Canvas(), event.AbsolutePosition)
}
