package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	sh "github.com/jeffwilliams/spacehoarder"
	dt "github.com/jeffwilliams/spacehoarder/dirtree"
	"github.com/jeffwilliams/spacehoarder/tree"
)

var (
	buildStatus  StatusPart
	deleteStatus StatusPart
	errorStatus  StatusPart
	statusLine   StatusLine
)

func init() {
	statusLine.Add(&buildStatus)
	statusLine.Add(&deleteStatus)
	statusLine.Add(&errorStatus)
}

type TcellPrintContext struct {
	View  views.View
	Style tcell.Style
	X, Y  int
}

func ViewPrint(ctx *TcellPrintContext, frmt string, args ...interface{}) (updatedCtx TcellPrintContext) {
	updatedCtx = *ctx

	str := fmt.Sprintf(frmt, args...)

	maxX, maxY := ctx.View.Size()

	// No wrapping supported, but \n starts a new line.

	x, y := ctx.X, ctx.Y

	defer func() {
		updatedCtx.X = x
		updatedCtx.Y = y
	}()

	if x < 0 {
		x = 0
	}
	if x >= maxX {
		return
	}
	if y < 0 {
		y = 0
	}
	if y >= maxY {
		return
	}

	for _, rn := range []rune(str) {
		if rn == '\n' {
			x = 0
			y++
		}
		if y >= maxY {
			return
		}
		if x >= maxX {
			continue // maybe a \n will occur
		}
		ctx.View.SetContent(x, y, rn, nil, ctx.Style)
		x++
	}

	return
}

type NodeData struct {
	flags TreeNodeFlags
	files []string
}

type TreeNodeFlags uint8

const (
	TreeNodeFlagExpanded TreeNodeFlags = 1 << iota
	// TreeNodeFlagVisible is false if an ancestor is not expanded.
	TreeNodeFlagHidden
	TreeNodeFlagFilesShown
)

var defaultTreeNodeFlags TreeNodeFlags

func treeNodeFlags(n *dt.Node) TreeNodeFlags {
	if n.UserData != nil {
		return n.UserData.(TreeNodeFlags)
	}
	return defaultTreeNodeFlags
}

func setTreeNodeFlags(n *dt.Node, f TreeNodeFlags) {
	n.UserData = f
}

func (t *TreeNodeFlags) Set(f TreeNodeFlags) TreeNodeFlags {
	*t = *t | f
	return *t
}

func (t *TreeNodeFlags) Unset(f TreeNodeFlags) TreeNodeFlags {
	*t = *t &^ f
	return *t
}

func (t TreeNodeFlags) IsSet(f TreeNodeFlags) bool {
	return (t & f) > 0
}

func SetTreeNodeFlag(n *dt.Node, f TreeNodeFlags) {
	flags := treeNodeFlags(n)
	setTreeNodeFlags(n, flags.Set(f))
}

func UnsetTreeNodeFlag(n *dt.Node, f TreeNodeFlags) {
	flags := treeNodeFlags(n)
	setTreeNodeFlags(n, flags.Unset(f))
}

func updateHiddenFlag(n *dt.Node) {
	// If any ancestor is collapsed, the node is hidden.
	hidden := false
	if n.Parent != nil {
		for n2 := n.Parent; n2 != nil; n2 = n2.Parent {
			if !treeNodeFlags(n2).IsSet(TreeNodeFlagExpanded) {
				hidden = true
				break
			}
		}
	}

	f := treeNodeFlags(n)
	if hidden {
		setTreeNodeFlags(n, f.Set(TreeNodeFlagHidden))
	} else {
		setTreeNodeFlags(n, f.Unset(TreeNodeFlagHidden))
	}
}

func updateHiddenFlagOnDescendants(n *dt.Node) {
	visitor := func(t tree.Tree, depth int) (cont bool) {
		cont = true
		if n == t {
			return
		}
		updateHiddenFlag(t.(*dt.Node))
		return
	}

	tree.Walk(n, visitor, tree.Forward, tree.PreOrder, n.Depth(), false)
}

type DirtreeWidget struct {
	views.WidgetWatchers
	dt   *dt.Dirtree
	view views.View
	// mutex protects dt.
	Mutex        sync.Mutex
	selectedNode *dt.Node
	selectedRow  int
	// first and last node in the window
	firstNode, lastNode *dt.Node
	screen              tcell.Screen
	ShowRoot            bool
	toDelete            *dt.Node
	savedStatus         string
	errStatus           StatusSetter
	delStatus           StatusSetter
	remove              chan *dt.Node
}

func NewDirtreeWidget(screen tcell.Screen, errStatus, delStatus StatusSetter) *DirtreeWidget {
	tree := dt.New()
	tree.SortChildren = true

	w := &DirtreeWidget{
		dt:        tree,
		screen:    screen,
		errStatus: errStatus,
		delStatus: delStatus,
		remove:    make(chan *dt.Node),
		//listeners: make(map[tcell.EventHandler]interface{}),
	}

	go w.remover()

	return w
}

func (w *DirtreeWidget) Draw() {
	w.draw()
}

func (w *DirtreeWidget) clampSelectedRow() {
	_, maxY := w.view.Size()
	if w.selectedRow < 0 {
		w.selectedRow = 0
	}
	if w.selectedRow >= maxY {
		w.selectedRow = maxY - 1
	}
}

func (w *DirtreeWidget) draw() {
	if w.dt == nil || w.view == nil {
		return
	}

	w.Mutex.Lock()
	defer w.Mutex.Unlock()

	if w.selectedNode == nil && w.dt.Root != nil {
		if w.ShowRoot {
			w.selectedNode = w.dt.Root
		} else if len(w.dt.Root.Children) > 0 {
			w.selectedNode = w.dt.Root.Child(0).(*dt.Node)
		}
	}

	if w.selectedNode == nil {
		return
	}

	debugOrigSelectedNode := w.selectedNode

	w.view.Clear()

	ctx := TcellPrintContext{
		View:  w.view,
		Style: tcell.StyleDefault,
		X:     0,
		Y:     0,
	}

	printNode := func(n *dt.Node, depth, y int) {
		ctx.X = 0
		ctx.Y = y
		sym := "+"
		if treeNodeFlags(n)&TreeNodeFlagExpanded > 0 {
			sym = "-"
		}
		if n.Info.Type == dt.PathTypeFile {
			sym = "F"
		}
		ctx = ViewPrint(&ctx, "%s%s ", strings.Repeat(" ", depth*2), sym)
		origStyle := ctx.Style
		ctx.Style = ctx.Style.Foreground(tcell.Color(172))
		acc := ""
		if !n.Info.SizeAccurate {
			acc = "?"
		}
		ctx = ViewPrint(&ctx, "[%s%s]", sh.FancySize(n.Info.Size), acc)
		ctx.Style = origStyle
		ViewPrint(&ctx, " %s", n.Info.Basename)
	}

	w.clampSelectedRow()

	w.firstNode = nil
	w.lastNode = nil

	y := w.selectedRow - 1
	delta := -1
	_, maxY := w.view.Size()

	visitor := func(t tree.Tree, depth int) (cont bool) {
		n := t.(*dt.Node)

		cont = true
		if treeNodeFlags(n)&TreeNodeFlagHidden > 0 {
			return
		}

		if n == w.dt.Root && !w.ShowRoot {
			return
		}

		if y < 0 || y >= maxY {
			cont = false
			return
		}

		if y == w.selectedRow {
			ctx.Style = ctx.Style.Background(tcell.ColorBlue)
		} else {
			ctx.Style = tcell.StyleDefault
		}

		printNode(n, depth, y)

		y += delta
		if delta < 0 {
			w.firstNode = n
		} else {
			w.lastNode = n
		}

		return
	}

	depth := w.selectedNode.Depth()
	if !w.ShowRoot {
		depth -= 1
	}

	tree.Walk(w.selectedNode, visitor, tree.Reverse, tree.PostOrder, depth, true)
	if w.firstNode == nil {
		// Nothing above selected row
		w.firstNode = w.selectedNode
	}

	y = w.selectedRow
	delta = 1

	tree.Walk(w.selectedNode, visitor, tree.Forward, tree.PreOrder, depth, false)

	if debugOrigSelectedNode != w.selectedNode {
		ctx.X = 0
		ctx.Y = 0
		ctx.Style = ctx.Style.Background(tcell.ColorRed)
		ctx = ViewPrint(&ctx, "[dbg] selectedNode changed.")
	}

}

func (w *DirtreeWidget) Resize() {
	log.Printf("DirtreeWidget.Resize called\n")
	w.WidgetWatchers.PostEventWidgetResize(w)
}

func (w *DirtreeWidget) nodeBelow() *dt.Node {
	nxt := tree.Next(w.selectedNode, tree.Forward, tree.PreOrder)
	for nxt != nil && treeNodeFlags(nxt.(*dt.Node))&TreeNodeFlagHidden > 0 {
		nxt = tree.Next(nxt, tree.Forward, tree.PreOrder)
	}

	if nxt == nil {
		return nil
	}
	return nxt.(*dt.Node)
}

func (w *DirtreeWidget) nodeAbove() *dt.Node {
	nxt := tree.Tree(w.selectedNode)
	for nxt != nil {
		prv := nxt
		nxt = tree.Next(nxt, tree.Reverse, tree.PostOrder)

		if nxt == nil && w.ShowRoot {
			nxt = prv
			break
		}

		if nxt.(*dt.Node) == w.dt.Root && !w.ShowRoot {
			nxt = prv
			break
		}

		if treeNodeFlags(nxt.(*dt.Node))&TreeNodeFlagHidden == 0 {
			break
		}
	}
	if nxt == nil {
		return nil
	}
	return nxt.(*dt.Node)
}

func (w *DirtreeWidget) refresh() {
	if w.selectedNode != nil {
		w.selectedNode.UpdateSize(0, true)
		w.selectedNode.DelAll()
		UnsetTreeNodeFlag(w.selectedNode, TreeNodeFlagFilesShown)
		build(w.screen, w, w.selectedNode, w.selectedNode.Info.Path, &dt.BuildOpts{IncludeFiles: false, OneFs: true}, nil)
	}
}

func (w *DirtreeWidget) toggleFiles() {
	if w.selectedNode != nil && w.selectedNode.Info.Type == dt.PathTypeDir {
		// Start a new set of goroutines that will build up the list of files under the
		// selected node.
		// Since we are recalculating the size, we set the current size to zero and let the
		// operations recalculate it.
		w.selectedNode.UpdateSize(0, true)
		w.selectedNode.DelAll()
		flags := treeNodeFlags(w.selectedNode)
		// toggle
		if flags.IsSet(TreeNodeFlagFilesShown) {
			onAdd := func(n *dt.Node) {
				UnsetTreeNodeFlag(n, TreeNodeFlagFilesShown)
			}
			build(w.screen, w, w.selectedNode, w.selectedNode.Info.Path, &dt.BuildOpts{IncludeFiles: false, OneFs: true}, onAdd)
			UnsetTreeNodeFlag(w.selectedNode, TreeNodeFlagFilesShown)
		} else {
			onAdd := func(n *dt.Node) {
				SetTreeNodeFlag(n, TreeNodeFlagFilesShown)
			}
			build(w.screen, w, w.selectedNode, w.selectedNode.Info.Path, &dt.BuildOpts{IncludeFiles: true, OneFs: true}, onAdd)
			SetTreeNodeFlag(w.selectedNode, TreeNodeFlagFilesShown)
		}
	}
}

func (w *DirtreeWidget) toggleExpanded() {
	if w.selectedNode != nil {
		w.Mutex.Lock()
		flags := treeNodeFlags(w.selectedNode)
		if flags.IsSet(TreeNodeFlagExpanded) {
			UnsetTreeNodeFlag(w.selectedNode, TreeNodeFlagExpanded)
		} else {
			SetTreeNodeFlag(w.selectedNode, TreeNodeFlagExpanded)
		}
		updateHiddenFlagOnDescendants(w.selectedNode)
		w.Mutex.Unlock()
	}
}

func (w *DirtreeWidget) selectNext() {
	if w.selectedNode != nil {
		w.Mutex.Lock()

		nxt := w.nodeBelow()
		if nxt != nil && w.selectedNode != nxt {
			w.selectedNode = nxt
			w.selectedRow += 1
			w.clampSelectedRow()
		}
		w.Mutex.Unlock()
	}
}

func (w *DirtreeWidget) selectPrev() {
	if w.selectedNode != nil {
		w.Mutex.Lock()

		nxt := w.nodeAbove()
		if nxt != nil {
			w.selectedNode = nxt
			//setStatus("debug: set selected to %p %v (root is %p %v)", w.selectedNode, w.selectedNode, w.dt.Root, w.dt.Root)
		}
		w.selectedRow -= 1
		w.clampSelectedRow()
		w.Mutex.Unlock()
	}
}

func (w *DirtreeWidget) selectFirst() {
	w.Mutex.Lock()
	if w.dt.Root != nil {
		if w.ShowRoot {
			w.selectedNode = w.dt.Root
		} else {
			w.selectedNode = w.dt.Root.Child(0).(*dt.Node)
		}
		w.selectedRow = 0
	}
	w.Mutex.Unlock()

}
func (w *DirtreeWidget) selectLast() {
	w.Mutex.Lock()
	last := (*dt.Node)(nil)
	visitor := func(t tree.Tree, depth int) (cont bool) {
		last = t.(*dt.Node)
		if treeNodeFlags(last)&TreeNodeFlagHidden == 0 && last != w.selectedNode {
			w.selectedRow += 1
		}
		return true
	}
	tree.Walk(w.selectedNode, visitor, tree.Forward, tree.PreOrder, w.selectedNode.Depth()-1, false)
	w.Mutex.Unlock()
	w.clampSelectedRow()
	if last != nil {
		w.selectedNode = last
	}

}

// removeNodeAndPath removes the node from the tree and the path from the FS.
// If removing fails, it's put back into the tree.
func (w *DirtreeWidget) removeNodeAndPath(n *dt.Node) {
	// Remove the node from the tree, but then set the node's parent field back to
	// what it was. This is needed by the removed goroutine to later
	// re-attach the node if needed.
	w.Mutex.Lock()
	parent := n.Parent
	parent.Del(n)
	n.Parent = parent
	w.Mutex.Unlock()

	select {
	case w.remove <- n:
	default:
		// Deletion is in progress.
		w.delStatus.SetStatus("Deleting failed: deletion is already in progress")
		w.Mutex.Lock()
		n.Parent.Add(n)
		w.Mutex.Unlock()
	}
}

func (w *DirtreeWidget) remover() {
	for n := range w.remove {
		err := os.RemoveAll(n.Info.Path)
		if err != nil {
			w.Mutex.Lock()
			w.delStatus.SetStatus("Deleting failed: %v", err)
			n.Parent.Add(n)
			w.Mutex.Unlock()
			// Rebuild the node in case some but not all of the descendants were deleted.
			build(w.screen, w, n, n.Info.Path, &dt.BuildOpts{IncludeFiles: false, OneFs: true}, nil)
		}
	}
}

func (w *DirtreeWidget) HandleEvent(ev tcell.Event) bool {
	unstageDelete := func() {
		w.toDelete = nil
		w.delStatus.SetStatus("")
	}

	switch ev := ev.(type) {
	case *tcell.EventKey:
		handled := true
		switch ev.Key() {
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'Q', 'q':
				app.Quit()
			case 'F', 'f':
				w.toggleFiles()
			case 'R', 'r':
				w.refresh()
			case 'Y', 'y':
				if w.toDelete != nil {
					w.delStatus.SetStatus("")
					//parent := w.selectedNode.Parent
					/*
						// Do delete
						// TODO: handle error somehow
						go os.RemoveAll(w.toDelete.Info.Path)
					*/
					n := w.selectedNode
					w.selectPrev()
					w.removeNodeAndPath(n)
					/*if parent != nil {
						w.selectedNode = parent
						w.refresh()
					} else {
						// Nothing left?
						w.selectedNode = nil
					}*/
				}
			default:
				handled = false
			}
		case tcell.KeyDown:
			w.selectNext()
		case tcell.KeyUp:
			w.selectPrev()
		case tcell.KeyCR:
			w.toggleExpanded()
		case tcell.KeyHome:
			w.selectFirst()
		case tcell.KeyEnd:
			w.selectLast()
		case tcell.KeyDelete:
			defer func() {
				w.toDelete = w.selectedNode
				w.delStatus.SetStatus("Type 'y' to confirm delete")
			}()
		default:
			handled = false
		}
		// User did not confirm delete
		unstageDelete()
		return handled

	case *DirtreeDrawEvent:
		return true
	case *DirtreeProgEvent:
		return true
	}

	return false
}

func (w *DirtreeWidget) SetView(view views.View) {
	w.view = view
}

func (w DirtreeWidget) Size() (int, int) {
	//return w.view.Size()
	/* We return the desired size as 0,0 here so that we take up
	available space in the parent panel (box layout). If we return
	the view size here then on resize we demand as much space as
	we were previously using which forces the panel to stay large
	(if we are resizing smaller) */
	return 0, 0
}
