package printers

type Printer interface {
	Print(data any) error
}
