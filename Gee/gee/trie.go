package gee

import "strings"

/**
 * 路由前缀树
 */
type node struct {
	// 待匹配路由，全集
	pattern string
	// 路由中的一部分，即单独一个路由段
	part string
	// 是否为模糊匹配，如*,:
	isWild   bool
	children []*node
}

// 返回第一个匹配成功的节点，用于插入
func (n *node) matchFirstChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}

	return nil
}

// 返回所有匹配成功的节点，用于查找
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)

	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}

	return nodes
}

// 插入节点
func (n *node) insert(pattern string, parts []string, location int) {
	// 最后一个part
	if len(parts) == location {
		// 将新的node赋值pattern
		n.pattern = pattern
		return
	}

	part := parts[location]
	child := n.matchFirstChild(part)

	// 当匹配到最后的part，如果不存在时，是一定会进入这里来创建一个新的node
	if child == nil {
		child = &node{part: part, isWild: part[0] == '*' || part[0] == ':'}
		n.children = append(n.children, child)
	}

	// 递归插入
	child.insert(pattern, parts, location + 1)
}

// 查找节点
func (n *node) search(parts []string, location int) *node {
	// 匹配到最后了或者存在模糊匹配时，结束查找
	if len(parts) == location || strings.HasPrefix(n.part, "*") {
		/**
		 * /p/:lang/doc只有在第三层节点，即doc节点，pattern才会设置为/p/:lang/doc。p和:lang节点的pattern属性皆为空。
		 * 因此，当匹配结束时，我们可以使用n.pattern == ""来判断路由规则是否匹配成功。
		 */
		if n.pattern == "" {
			return nil
		}
		return n
	}

	part := parts[location]
	children := n.matchChildren(part)

	// dfs遍历每个节点查找
	for _, child := range children {
		result := child.search(parts, location + 1)
		if result != nil {
			return result
		}
	}

	return nil
}