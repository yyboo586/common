package common

import "unsafe"

/*
BenchmarkByteSlice2String1_Long-12     	55680343	        21.66 ns/op	      48 B/op	       1 allocs/op
BenchmarkByteSlice2String2_Long-12     	1000000000	         0.74 ns/op	       0 B/op	       0 allocs/op
BenchmarkByteSlice2String3_Long-12     	44969457	        23.36 ns/op	      24 B/op	       1 allocs/op
BenchmarkByteSlice2String4_Long-12     	1000000000	         0.47 ns/op	       0 B/op	       0 allocs/op
BenchmarkByteSlice2String1_Short-12    	111562826	        10.88 ns/op	       3 B/op	       1 allocs/op
BenchmarkByteSlice2String2_Short-12    	1000000000	         0.87 ns/op	       0 B/op	       0 allocs/op
BenchmarkByteSlice2String3_Short-12    	48466321	        23.47 ns/op	      24 B/op	       1 allocs/op
BenchmarkByteSlice2String4_Short-12    	1000000000	         0.46 ns/op	       0 B/op	       0 allocs/op
*/

func ByteSlice2String1(b []byte) (s string) {
	return string(b)
}

func ByteSlice2String2(b []byte) (s string) {
	return *(*string)(unsafe.Pointer(&b))
}

func ByteSlice2String3(b []byte) (s string) {
	return unsafe.String((*byte)(unsafe.Pointer(&b)), len(b))
}

func ByteSlice2String4(b []byte) (s string) {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
