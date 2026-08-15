package lsm

type MemTable struct {
	data map[string]*Entry
}

type Entry struct {
	value   string
	deleted bool
}

func (m *MemTable) Set(key, value string) {
	m.data[key] = &Entry{
		value:   value,
		deleted: false,
	}
}

func (m *MemTable) Get(key string) (string, bool) {
	entry, ok := m.data[key]
	if !ok || entry.deleted {
		return "", false
	}
	return entry.value, true
}

func (m *MemTable) Delete(key string) {
	m.data[key] = &Entry{
		deleted: true,
	}
}

func (m *MemTable) Size() int {
	return len(m.data)
}
