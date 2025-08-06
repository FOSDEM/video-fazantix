package theatre

type EventListener func(theatre *Theatre, data interface{})

type EventDataSetScene struct {
	Event string
	Stage string
	Scene string
}

type EventTallyData struct {
	Stage string
	Tally map[string]bool
}

func (t *Theatre) AddEventListener(event string, callback EventListener) {
	t.listener[event] = append(t.listener[event], callback)
}

func (t *Theatre) invoke(event string, data interface{}) {
	for _, listener := range t.listener[event] {
		go listener(t, data)
	}
}
