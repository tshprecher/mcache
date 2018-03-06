package store

// kvSize returns the size in bytes of the key and value data
func kvSize(key string, val Value) int {
	return len(key) + len(val.Bytes) + 2 /* flags */ + 8 /* cas unique */
}

// kvListNode represents a doubly linked list node of key-value pairs
type kvListNode struct {
	key        string
	val        Value
	prev, next *kvListNode
}

// EvictionPolicy defines the strategy by which a StorageEngine
// should evict Values.
type EvictionPolicy interface {
	// Capacity returns the total capacity, in bytes, of the memory to manage.
	Capacity() int

	// Used returns the current memory used, in bytes. The size of keys is included.
	Used() int

	// Touch indicates that a key has been written or read
	Touch(key string) bool

	// Add returns the set of keys that should be removed from the
	// in-memory store and a boolean indicating if there is room to
	// store the input value.
	Add(key string, v Value) (evict []string, hasSpace bool)

	// Remove indicates that the key has been removed from the StorageEngine.
	// It returns false if and only if the EvictionPolicy is not managing
	// the key.
	Remove(key string) bool
}

// A lruEvictionPolicy is an LRU implementation of an EvictionPolicy
// using doubly linked lists.
type lruEvictionPolicy struct {
	cap      int
	used     int
	sentinel *kvListNode
	kvMap    map[string]*kvListNode
}

func NewLruEvictionPolicy(cap int) *lruEvictionPolicy {
	if cap < 0 {
		cap = 0
	}
	s := &kvListNode{}
	s.next = s
	s.prev = s
	return &lruEvictionPolicy{cap, 0, s, map[string]*kvListNode{}}
}

func (l *lruEvictionPolicy) Capacity() int {
	return l.cap
}

func (l *lruEvictionPolicy) Used() int {
	return l.used
}

func (l *lruEvictionPolicy) Touch(key string) bool {
	node, ok := l.kvMap[key]
	if !ok {
		return false
	}

	// pop the node and bring it to the front
	node.prev.next = node.next
	node.next.prev = node.prev
	node.next = l.sentinel.next
	node.prev = l.sentinel
	l.sentinel.next.prev = node
	l.sentinel.next = node
	return true
}

func (l *lruEvictionPolicy) Add(key string, v Value) (evict []string, hasSpace bool) {
	size := kvSize(key, v)
	if size > l.cap {
		hasSpace = false
		return
	}
	hasSpace = true

	existing, ok := l.kvMap[key]
	existingSize := 0
	if ok {
		existingSize = kvSize(key, existing.val)
	}

	// add evictions, if necessary
	for l.used+size-existingSize > l.cap {
		// evict from the end of the list (least recent)
		lruNode := l.sentinel.prev
		l.used -= kvSize(lruNode.key, lruNode.val)
		delete(l.kvMap, lruNode.key)
		l.sentinel.prev = lruNode.prev
		l.sentinel.prev.next = l.sentinel
		evict = append(evict, lruNode.key)
	}

	// finally, add the new node to the front of the list (most recent)
	node := &kvListNode{key, v, nil, nil}
	node.next = l.sentinel.next
	node.prev = l.sentinel
	node.next.prev = node
	l.sentinel.next = node
	l.kvMap[key] = node
	l.used += size - existingSize

	return
}

func (l *lruEvictionPolicy) Remove(key string) bool {
	node, ok := l.kvMap[key]
	if !ok {
		return false
	}
	delete(l.kvMap, key)
	node.prev.next = node.next
	node.next.prev = node.prev
	return true
}
