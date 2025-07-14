package theatre

type EventListener func(theatre *Theatre, data interface{})

type EventDataSetScene struct {
	Event string
	Stage string
	Scene string
}

func (t *Theatre) AddEventListener(event string, callback EventListener) {
	t.listener[event] = append(t.listener[event], callback)
}

func (t *Theatre) invoke(event string, data interface{}) {
	for _, listener := range t.listener[event] {
		go listener(t, data)
	}
}
