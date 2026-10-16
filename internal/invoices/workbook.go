package invoices

import "sitewise/internal/xlsx"

// The ledger export writes through the shared workbook writer.
type (
	Cell  = xlsx.Cell
	Sheet = xlsx.Sheet
)

var (
	WriteXLSX  = xlsx.WriteXLSX
	ColumnName = xlsx.ColumnName
)
