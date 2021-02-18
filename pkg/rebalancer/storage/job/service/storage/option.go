package storage

type Option func(b *bs) error

var defaultOptions = []Option{}
