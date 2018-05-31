package main

import (
	"bytes"
	"fmt"
)

type TextSetter interface {
	SetText(string)
}

type StatusLine struct {
	parts  []StatusPart
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

func (l *StatusLine) Add(p StatusPart) {
	l.parts = append(l.parts, p)
	p.OnChange(l)
}

func (l *StatusLine) changed() {
	if l.setter != nil {
		l.setter.SetText(l.String())
	}
}

type Changer interface {
	changed()
}

type StatusPart interface {
	fmt.Stringer
	OnChange(c Changer)
}

type statusPart struct {
	text     string
	c        Changer
	brackets bool
}

type StatusSetter interface {
	SetStatus(s string, args ...interface{})
}

func (p *statusPart) SetStatus(s string, args ...interface{}) {
	p.text = fmt.Sprintf(s, args...)
	if p.c != nil {
		p.c.changed()
	}
}

func (p *statusPart) String() string {
	if p.brackets {
		return fmt.Sprintf("[%s]", p.text)
	} else {
		return p.text
	}
}

func (p *statusPart) OnChange(c Changer) {
	p.c = c
}
