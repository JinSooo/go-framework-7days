package lru

import (
	"fmt"
	"reflect"
	"testing"
)

type String string

func (s String) Len() int {
	return len(s)
}

func TestGet(t *testing.T) {
	lru := New(int64(0), nil)
	lru.Set("key1", String("1234"))

	if ele, ok := lru.Get("key1"); !ok || string(ele.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache hit key2 failed")
	}
}

func TestRemoveOldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + k2 + v1 + v2)

	lru := New(int64(cap), nil)
	lru.Set(k1, String(v1))
	lru.Set(k2, String(v2))
	lru.Set(k3, String(v3))

	if _, ok := lru.Get(k1); ok || lru.Len() != 2 {
		t.Fatalf("cache removeOldest key1 failed")
	}
}

func TestCallback(t *testing.T) {
	keys := make([]string, 0)
	var callback callback = func(key string, value Value) {
		keys = append(keys, key)
		fmt.Printf("key: %v, value: %v", key, value)
	}

	lru := New(int64(10), callback)
	lru.Set("k22", String("v22"))
	lru.Set("k33", String("v33"))
	lru.Set("k4", String("k4"))

	if !reflect.DeepEqual(keys, []string{"k22"}) {
		t.Fatalf("call callback failed, keys: %s", keys)
	}
}
