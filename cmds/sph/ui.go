package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	sh "github.com/jeffwilliams/spacehoarder"
	dt "github.com/jeffwilliams/spacehoarder/dirtree"
	"github.com/jeffwilliams/spacehoarder/tree"
)

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

type TreeNodeFlags uint8

const (
	TreeNodeFlagExpanded TreeNodeFlags = 1 << iota
	// TreeNodeFlagVisible is false if an ancestor is not expanded.
	TreeNodeFlagHidden
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
}

func NewDirtreeWidget(screen tcell.Screen) *DirtreeWidget {
	dt := dt.New()
	dt.SortChildren = true
	return &DirtreeWidget{
		dt: dt,
		//listeners: make(map[tcell.EventHandler]interface{}),
	}
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

	if w.selectedNode == nil && w.dt.Root != nil && len(w.dt.Root.Children) > 0 {
		w.selectedNode = w.dt.Root.Child(0).(*dt.Node)
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
		ctx = ViewPrint(&ctx, "%s%s ", strings.Repeat(" ", depth*2), sym)
		origStyle := ctx.Style
		ctx.Style = ctx.Style.Foreground(tcell.Color(172))
		acc := ""
		if !n.Dir.SizeAccurate {
			acc = "?"
		}
		ctx = ViewPrint(&ctx, "[%s%s]", sh.FancySize(n.Dir.Size), acc)
		ctx.Style = origStyle
		ViewPrint(&ctx, " %s", n.Dir.Basename)
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

		if n == w.dt.Root {
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

	tree.Walk(w.selectedNode, visitor, tree.Reverse, tree.PostOrder, w.selectedNode.Depth()-1, true)
	if w.firstNode == nil {
		// Nothing above selected row
		w.firstNode = w.selectedNode
	}

	y = w.selectedRow
	delta = 1

	tree.Walk(w.selectedNode, visitor, tree.Forward, tree.PreOrder, w.selectedNode.Depth()-1, false)

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

		if nxt.(*dt.Node) == w.dt.Root {
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

func (w *DirtreeWidget) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'Q', 'q':
				app.Quit()
				return true
			}
		case tcell.KeyDown:
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
			return true
		case tcell.KeyUp:
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
			return true
		case tcell.KeyCR:
			if w.selectedNode != nil {
				w.Mutex.Lock()
				flags := treeNodeFlags(w.selectedNode)
				if flags.IsSet(TreeNodeFlagExpanded) {
					setTreeNodeFlags(w.selectedNode, flags.Unset(TreeNodeFlagExpanded))
				} else {
					setTreeNodeFlags(w.selectedNode, flags.Set(TreeNodeFlagExpanded))
				}
				updateHiddenFlagOnDescendants(w.selectedNode)
				w.Mutex.Unlock()
			}
			return true

		case tcell.KeyHome:
			w.Mutex.Lock()
			if w.dt.Root != nil {
				w.selectedNode = w.dt.Root.Child(0).(*dt.Node)
				w.selectedRow = 0
			}
			w.Mutex.Unlock()
		case tcell.KeyEnd:
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
