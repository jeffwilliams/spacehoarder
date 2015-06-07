package dirtree

import (
	"encoding/gob"
	"fmt"
	"io"
)

func Encode(opsW io.Writer, progW io.Writer, ops chan OpData, prog chan string) {
	opEnc := gob.NewEncoder(opsW)
	progEnc := gob.NewEncoder(progW)

	//ops, prog := dirtree.Build(flag.Arg(0))

loop:
	for {
		select {
		case op, ok := <-ops:
			if !ok {
				ops = nil
				continue loop
			}
			fmt.Println("Sending op", op)
			opEnc.Encode(op)

		case f, ok := <-prog:
			if !ok {
				prog = nil
				continue loop
			}

			progEnc.Encode(f)
		}

		if ops == nil && progEnc == nil {
			break loop
		}
	}
}

func Decode(opsR io.Reader, progR io.Reader, ops chan OpData, prog chan string) {
	opDec := gob.NewDecoder(opsR)
	progDec := gob.NewDecoder(progR)

	go func() {
		defer close(ops)

		for {
			// The gob decoder seems to only fill in fields that are set to the zero value,
			// So each pass we create a fresh OpData struct.
			op := OpData{}
			if err := opDec.Decode(&op); err != nil {
				fmt.Println("Error decoding op:", err)
				return
			}

			fmt.Println("Got op: ", op)

			ops <- op
		}
	}()

	go func() {
		defer close(prog)

		var f string
		for {
			if err := progDec.Decode(&f); err != nil {
				fmt.Println("Error decoding progress:", err)
				return
			}

			prog <- f
		}
	}()
}
