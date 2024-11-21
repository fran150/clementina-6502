package buses

type StandaloneLine struct {
	status bool
}

func CreateStandaloneLine(status bool) *StandaloneLine {
	return &StandaloneLine{
		status: status,
	}
}

func (line *StandaloneLine) Status() bool {
	return line.status
}

func (line *StandaloneLine) Set(value bool) {
	line.status = value
}

func (line *StandaloneLine) Toggle() {
	line.Set(!line.Status())
}
