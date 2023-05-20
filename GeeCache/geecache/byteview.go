package geecache

/* -------------------------------- 缓存值的抽象与封装 ------------------------------- */
/**
 * 实现 Value 接口
 *
 * ByteView 只有一个数据成员，b []byte，b 将会存储真实的缓存值。
 * 选择 byte 类型是为了能够支持任意的数据类型的存储
 */

type ByteView struct {
	b []byte
}

// 返回长度
func (byteView ByteView) Len() int {
	return len(byteView.b)
}

// 返回字符串
func (byteView ByteView) String() string {
	return string(byteView.b)
}

// 使用 ByteSlice() 方法返回一个拷贝，防止缓存值被外部程序修改
func (byteView ByteView) ByteSlice() []byte {
	return cloneBytes(byteView.b)
}

// 拷贝byte
func cloneBytes(b []byte) []byte {
	clone := make([]byte, len(b))
	copy(clone, b)
	return clone
}
