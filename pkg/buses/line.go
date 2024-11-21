package buses

type Line interface {
	Status() bool
	Set(value bool)
	Toggle()
}
