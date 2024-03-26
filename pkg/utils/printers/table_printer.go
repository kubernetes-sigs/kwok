/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package printers

import (
	"io"
)

// TablePrinter is a utility for printing tables.
// It maintains a list of column widths to ensure that the table is formatted correctly.
type TablePrinter struct {
	widths []int
	w      io.Writer
}

// NewTablePrinter creates a new TablePrinter that writes to the provided writer.
func NewTablePrinter(w io.Writer) *TablePrinter {
	return &TablePrinter{w: w}
}

func (p *TablePrinter) setWidths(records []string) {
	if len(p.widths) < len(records) {
		widths := make([]int, len(records))
		copy(widths, p.widths)
		p.widths = widths
	}

	for i, record := range records {
		if p.widths[i] < len(record) {
			p.widths[i] = len(record)
		}
	}
}

// Write writes a single row to the table.
// It adjusts the column widths as necessary to ensure that the table is formatted correctly.
// Each cell in the row is padded with spaces to match the width of the widest cell in the column.
func (p *TablePrinter) Write(records []string) error {
	p.setWidths(records)

	return p.write(records)
}

// WriteAll writes multiple rows to the table.
// It first adjusts the column widths based on all the rows, then writes each row to the table.
func (p *TablePrinter) WriteAll(records [][]string) error {
	for _, records := range records {
		p.setWidths(records)
	}

	for _, records := range records {
		err := p.write(records)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *TablePrinter) write(records []string) error {
	for i, record := range records {
		_, err := p.w.Write([]byte(record))
		if err != nil {
			return err
		}

		if i != len(records)-1 {
			err = writeSpaces(p.w, p.widths[i]-len(record)+3)
			if err != nil {
				return err
			}
		}
	}

	_, err := p.w.Write([]byte{'\n'})
	if err != nil {
		return err
	}
	return nil
}

func writeSpaces(w io.Writer, n int) error {
	for i := 0; i < n; i++ {
		if _, err := w.Write([]byte{' '}); err != nil {
			return err
		}
	}
	return nil
}
