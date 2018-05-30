package main

import (
	"bytes"
	"fmt"
)

type TextSetter interface {
	SetText(string)
}

type StatusLine struct {
	parts  []*StatusPart
	setter TextSetter
}

func (l StatusLine) String() string {
	var buf bytes.Buffer
	for i, p := range l.parts {
		if i != 0 {
			buf.WriteRune(' ')
		}
		buf.WriteString(p.String())
	}
	return buf.String()
}

func (l *StatusLine) Add(p *StatusPart) {
	l.parts = append(l.parts, p)
	p.parent = l
}

func (l *StatusLine) changed() {
	if l.setter != nil {
		l.setter.SetText(l.String())
	}
}

type StatusPart struct {
	text   string
	parent *StatusLine
}

func (p *StatusPart) SetStatus(s string, args ...interface{}) {
	p.text = fmt.Sprintf(s, args...)
	if p.parent != nil {
		p.parent.changed()
	}
}

func (p *StatusPart) String() string {
	return p.text
}

type StatusSetter interface {
	SetStatus(s string, args ...interface{})
}
