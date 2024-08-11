package provider

type ChangeFunc func() error

type Provider interface {
	Watch() error
	SetOnChanged(ChangeFunc)
}
