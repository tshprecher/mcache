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

type EvictionPolicy interface {
	Capacity() int
	Used() int
	Touch(key string) bool
	Add(key string, v Value) (evict []string, hasSpace bool)
}

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
		l.sentinel.prev = l.sentinel
		evict = append(evict, lruNode.key)
	}

	// finally, add the new node to the front of the list (most recent)
	node := &kvListNode{key, v, nil, nil}
	l.sentinel.next.prev = node
	node.next = l.sentinel.next
	node.prev = l.sentinel
	l.sentinel = node
	l.kvMap[key] = node
	l.used += size - existingSize

	return
}
