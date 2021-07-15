package domain

type (
	NamedAddress struct {
		Name string
		Addr string
	}
)

func NewNamedAddress(n, a string) NamedAddress {
	return NamedAddress{
		Name: n,
		Addr: a,
	}
}
