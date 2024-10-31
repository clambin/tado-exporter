package oapi

// VarP returns a pointer to the value. This is used to create pointers to constants whn dealing with generated openapi clients.
func VarP[T any](t T) *T {
	return &t
}
