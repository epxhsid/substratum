package lsm

type MemTable struct {
	data map[string]*Entry
	size int
}

type Entry struct {
	value   string
	deleted bool
}

func NewMemTable() *MemTable {
	return &MemTable{
		data: make(map[string]*Entry),
	}
}

func (m *MemTable) Set(key, value string) {
	m.ensureInit()
	if old, exists := m.data[key]; exists {
		m.size -= 9 + len(key) + len(old.value)
	}

	entry := &Entry{
		value: value,
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
	if old, exists := m.data[key]; exists {
		m.size -= 9 + len(key) + len(old.value)
	}

	entry := &Entry{
		deleted: true,
	}

	m.data[key] = entry
	m.size += 9 + len(key) + len(entry.value)
}

func (m *MemTable) Size() int {
	return m.size
}

func (m *MemTable) Length() int {
	return len(m.data)
}

func (m *MemTable) ensureInit() {
	if m.data == nil {
		m.data = make(map[string]*Entry)
	}
}
