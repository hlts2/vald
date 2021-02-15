package service

type StorageOption func(b *bs) error

var defaultStorageOptions = []StorageOption{}
