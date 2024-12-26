package common

import (
	"reflect"
	"unsafe"
)

/*
BenchmarkString2ByteSlice1_long-12     	48168222	        25.16 ns/op	      48 B/op	       1 allocs/op
BenchmarkString2ByteSlice2_long-12     	524661426	         2.27 ns/op	       0 B/op	       0 allocs/op
BenchmarkString2ByteSlice3_long-12     	1000000000	         0.68 ns/op	       0 B/op	       0 allocs/op
BenchmarkString2ByteSlice1_Short-12    	69356744	        16.05 ns/op	       8 B/op	       1 allocs/op
BenchmarkString2ByteSlice2_Short-12    	527904861	         2.25 ns/op	       0 B/op	       0 allocs/op
BenchmarkString2ByteSlice3_Short-12    	1000000000	         0.67 ns/op	       0 B/op	       0 allocs/op
*/

func String2ByteSlice1(s string) (b []byte) {
	return append(b, s...)
}

func String2ByteSlice2(s string) (b []byte) {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))

	bh.Data = sh.Data
	bh.Len = sh.Len
	bh.Cap = sh.Len

	return
}

func String2ByteSlice3(s string) (b []byte) {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
