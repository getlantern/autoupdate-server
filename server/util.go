package server

import "context"

const userIDKey = "UserId"

func userIDFromRequest(c context.Context) int64 {
	if id := c.Value(userIDKey); id != nil {
		return id.(int64)
	}
	return 0
}
