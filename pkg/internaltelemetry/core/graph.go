package core


func [T any] dfs(consumer func(prev *Vertex[T], cur *Vertex[T]))