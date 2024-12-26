package common

import "testing"

var result []byte // 避免编译器优化

func BenchmarkString2ByteSlice1_long(b *testing.B) {
	s := "这是一个测试字符串for benchmarking"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = String2ByteSlice1(s)
	}
}

func BenchmarkString2ByteSlice2_long(b *testing.B) {
	s := "这是一个测试字符串for benchmarking"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = String2ByteSlice2(s)
	}
}

func BenchmarkString2ByteSlice3_long(b *testing.B) {
	s := "这是一个测试字符串for benchmarking"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = String2ByteSlice3(s)
	}
}

// 添加不同长度字符串的测试
func BenchmarkString2ByteSlice1_Short(b *testing.B) {
	s := "abc"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = String2ByteSlice1(s)
	}
}

func BenchmarkString2ByteSlice2_Short(b *testing.B) {
	s := "abc"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = String2ByteSlice2(s)
	}
}

func BenchmarkString2ByteSlice3_Short(b *testing.B) {
	s := "abc"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = String2ByteSlice3(s)
	}
}
