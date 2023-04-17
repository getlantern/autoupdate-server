package instrument

import "context"

func FromContext(ctx context.Context) context.Context {
	return fromContext(ctx, otelContextKey)
}

func UserContext(ctx context.Context) context.Context {
	return fromContext(ctx, userContextKey)
}

func fromContext(ctx context.Context, k string) context.Context {
	if v := ctx.Value(k); v != nil {
		return v.(context.Context)
	}
	return nil
}

func Get(ctx context.Context, k any) any {
	if v := ctx.Value(k); v != nil {
		return v
	}
	return nil
}

func Value(ctx context.Context, k any) string {
	if v := Get(ctx, k); v != nil {
		return v.(string)
	}
	return ""
}
