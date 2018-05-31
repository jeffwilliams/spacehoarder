package main

import (
	"sync"
	"time"

	"github.com/gdamore/tcell"
	sh "github.com/jeffwilliams/spacehoarder"
	dt "github.com/jeffwilliams/spacehoarder/dirtree"
)

func ApplyAll(screen tcell.Screen, t *dt.Dirtree, root *dt.Node, m *sync.Mutex, ops chan dt.OpData, onAdd WhenNodeAdded) {

	ch := make(chan struct{})

	go func() {
		for _ = range ch {
			de := DirtreeDrawEvent(time.Now())
			screen.PostEvent(&de)
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	ctx := dt.NewApplyContext(root)

	for op := range ops {
		m.Lock()
		added := t.ApplyCtx(ctx, op)
		if added != nil {
			updateHiddenFlag(added)
			buildStatus.SetStatus("Processing %s", added.Info.Path)
		}
		if added == t.Root {
			// Root node is always expanded
			setTreeNodeFlags(t.Root, treeNodeFlags(t.Root)|TreeNodeFlagExpanded)
		}
		m.Unlock()

		select {
		case ch <- struct{}{}:
		default:
		}
	}

	if t.Root != nil {
		buildStatus.SetStatus("Total %s", sh.FancySize(t.Root.Info.Size))
	} else {
		buildStatus.SetStatus(".")
	}
	de := DirtreeDrawEvent(time.Now())
	screen.PostEvent(&de)
}

func drop(c chan string) {
	for _ = range c {

	}
}

type WhenNodeAdded func(n *dt.Node)

// build sets up pipelines used to add nodes to the
// dirtree that we display in the ui.
func build(screen tcell.Screen, dtw *DirtreeWidget, rootNode *dt.Node, rootPath string, opts *dt.BuildOpts, onAdd WhenNodeAdded) {
	if opts == nil {
		opts = dt.DefaultBuildOpts
	}
	ops, prog := dt.Build(rootPath, opts)
	go ApplyAll(screen, dtw.dt, rootNode, &dtw.Mutex, ops, onAdd)
	go drop(prog)
}
