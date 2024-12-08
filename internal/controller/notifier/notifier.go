package notifier

type Notifier interface {
	Notify(string)
}

type Notifiers []Notifier

func (n Notifiers) Notify(msg string) {
	for _, l := range n {
		l.Notify(msg)
	}
}
