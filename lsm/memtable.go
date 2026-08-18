package lsm

type MemTable struct {
	data map[string]*Entry
}

type Entry struct {
	value   string
	deleted bool
}

func (m *MemTable) ensureInit() {
	if m.data == nil {
		m.data = make(map[string]*Entry)
	}
}

func (m *MemTable) Set(key, value string) {
	m.ensureInit()
	m.data[key] = &Entry{
		value:   value,
		deleted: false,
	}
}

func (m *MemTable) Get(key string) (string, bool) {
	if m.data == nil {
		return "", false
	}
	entry, ok := m.data[key]
	if !ok || entry.deleted {
		return "", false
	}
	return entry.value, true
}

func (m *MemTable) Delete(key string) {
	m.ensureInit()
	m.data[key] = &Entry{
		deleted: true,
	}
}

func (m *MemTable) Size() int {
	if m.data == nil {
		return 0
	}
	return len(m.data)
}
