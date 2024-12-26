package common

import "testing"

// 避免编译器优化
var resultStr string

func BenchmarkByteSlice2String1_Long(b *testing.B) {
	data := []byte("这是一个测试字符串for benchmarking")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStr = ByteSlice2String1(data)
	}
}

func BenchmarkByteSlice2String2_Long(b *testing.B) {
	data := []byte("这是一个测试字符串for benchmarking")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStr = ByteSlice2String2(data)
	}
}

func BenchmarkByteSlice2String3_Long(b *testing.B) {
	data := []byte("这是一个测试字符串for benchmarking")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStr = ByteSlice2String3(data)
	}
}

func BenchmarkByteSlice2String4_Long(b *testing.B) {
	data := []byte("这是一个测试字符串for benchmarking")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStr = ByteSlice2String4(data)
	}
}

func BenchmarkByteSlice2String1_Short(b *testing.B) {
	data := []byte("abc")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStr = ByteSlice2String1(data)
	}
}

func BenchmarkByteSlice2String2_Short(b *testing.B) {
	data := []byte("abc")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStr = ByteSlice2String2(data)
	}
}

func BenchmarkByteSlice2String3_Short(b *testing.B) {
	data := []byte("abc")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStr = ByteSlice2String3(data)
	}
}

func BenchmarkByteSlice2String4_Short(b *testing.B) {
	data := []byte("abc")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStr = ByteSlice2String4(data)
	}
}
