package benchmark

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
)

/*
goos: linux
goarch: amd64
pkg: github.com/yyboo586/common/benchmark
cpu: AMD Ryzen 5 5600G with Radeon Graphics
BenchmarkUnmarshal-12            	 4161603	       325.0 ns/op	     384 B/op	      11 allocs/op
BenchmarkUnmarshalWithPool-12    	 4110174	       294.1 ns/op	     304 B/op	      10 allocs/op
BenchmarkBuffer-12               	  989620	        1059 ns/op	   10240 B/op	       1 allocs/op
BenchmarkBufferWithPool-12       	58258051	       19.36 ns/op	       0 B/op	       0 allocs/op
PASS
*/

// sync.Pool: 保存和复用临时对象，减少内存分配，降低GC压力

var userPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(User)
	},
}

type User struct {
	Name  string
	Age   int
	Sex   int8
	Email string
	Array []int
}

func (u *User) reset() {
	u.Name = ""
	u.Age = 0
	u.Sex = 0
	u.Email = ""
	u.Array = nil
}

var buf, _ = json.Marshal(User{"tom", 18, 0, "tom@gmail.com", []int{1, 2, 3}})

func BenchmarkUnmarshal(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			user := &User{}
			_ = json.Unmarshal(buf, user)
		}
	})
}

func BenchmarkUnmarshalWithPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			user := userPool.Get().(*User)
			_ = json.Unmarshal(buf, user)
			user.reset()
			userPool.Put(user)
		}
	})
}

var bufferPool sync.Pool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

var data []byte = make([]byte, 10000)

func BenchmarkBuffer(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buffer bytes.Buffer
			buffer.Write(data)
		}
	})
}

func BenchmarkBufferWithPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer := bufferPool.Get().(*bytes.Buffer)
			buffer.Write(data)
			buffer.Reset()
			bufferPool.Put(buffer)
		}
	})
}
