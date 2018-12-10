package dfudevice

type Progress interface {
	Reset()
	Increment()
	SetStatus(string)
	SetIncrement(uint)
	SetMax(uint)
}

type progressList struct {
	list []Progress
}

func (p *progressList) add(progress Progress) {
	p.list = append(p.list, progress)
}

func (p *progressList) reset() {
	for _, progress := range p.list {
		progress.Reset()
	}
}

func (p *progressList) increment() {
	for _, progress := range p.list {
		progress.Increment()
	}
}

func (p *progressList) setStatus(status string) {
	for _, progress := range p.list {
		progress.SetStatus(status)
	}
}

func (p *progressList) setIncrement(inc uint) {
	for _, progress := range p.list {
		progress.SetIncrement(inc)
	}
}

func (p *progressList) setMax(max uint) {
	for _, progress := range p.list {
		progress.SetMax(max)
	}
}
