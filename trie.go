package main

type node struct {
	pattern  string  // 传入的字符串。
	part     string  // 路由的一部分
	children []*node // 传入的子节点 .. 只不过是字符串。并没有到树的程度。。
	isWild   bool
}

func (n *node) matchChild(part string) *node {
	for _, v := range n.children {
		//  是否为精确匹配
		if v.part == part || v.isWild {
			return n
		}
	}
	return nil
}

// 查找所有的节点。
func (n *node) matchChildren(part string) []*node {
	r := make([]*node, 0)
	for _, v := range n.children {
		if v.part == part || v.isWild {
			r = append(r, v)
		}
	}
	return r
}

// 插入的高度和宽度
func (n *node) insert(pattern string, parts []string, height int) {
	// 匹配字符串，
	//  如果是部分的长度和高度重合的话
	if len(parts) == height {
		n.pattern = pattern
		return
	}
	part := parts[height]
	// 找到部分的孩子树
	child := n.matchChild(part)
	// 如果有没有这个树的话
	if child == nil {
		// 下一个子节点。。。
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		//  添加到這個節點里面
		n.children = append(n.children, child)
	}
	// 直接添加到里面，用递归的方式
	child.insert(pattern, parts, height+1)
}

func (n *node) search(parts []string, height int) *node {
	if len(parts) == height {
		if n.pattern == "" { // 匹配到节点。 直接返回
			return nil
		}
		return n
	}
	part := parts[height]             // 找到部分。
	children := n.matchChildren(part) // 所有的子節點

	for _, child := range children {
		r := child.search(parts, height+1)
		if r != nil { // 如果有子节点话，直接返回
			return r
		}
	}
	return nil
}
