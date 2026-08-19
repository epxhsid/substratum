package lsm

type MemTable struct {
	data map[string]*Entry
	size int
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

	old, exists := m.data[key]
	if exists {
		m.size -= 9 + len(key) + len(old.value)
	}

	entry := &Entry{
		value:   value,
		deleted: false,
	}

	m.data[key] = entry
	m.size += 9 + len(key) + len(value)
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

	old, exists := m.data[key]
	if exists {
		m.size -= 9 + len(key) + len(old.value)
	}

	entry := &Entry{
		deleted: true,
	}

	m.data[key] = entry
	m.size += 9 + len(key)
}

func (m *MemTable) Size() int {
	if m.data == nil {
		return 0
	}
	return len(m.data)
}
