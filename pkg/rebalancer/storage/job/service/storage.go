package service

import (
	"context"
	"io"
	"reflect"

	"github.com/vdaas/vald/internal/compress"
	"github.com/vdaas/vald/internal/config"
	"github.com/vdaas/vald/internal/db/storage/blob"
	"github.com/vdaas/vald/internal/db/storage/blob/s3"
	"github.com/vdaas/vald/internal/db/storage/blob/s3/session"
	"github.com/vdaas/vald/internal/errgroup"
	"github.com/vdaas/vald/internal/errors"
)

type Storage interface {
	Reader(ctx context.Context) (io.ReadCloser, error)
}

type bs struct {
	eg          errgroup.Group
	storageType string
	bucketName  string
	filename    string
	suffix      string

	s3Opts        []s3.Option
	s3SessionOpts []session.Option

	compressAlgorithm string
	compressionLevel  int

	bucket     blob.Bucket
	compressor compress.Compressor
}

func NewStorage(opts ...StorageOption) (Storage, error) {
	b := new(bs)
	for _, opt := range append(defaultStorageOptions, opts...) {
		if err := opt(b); err != nil {
			return nil, errors.ErrOptionFailed(err, reflect.ValueOf(opt))
		}
	}

	err := b.initCompressor()
	if err != nil {
		return nil, err
	}

	err = b.initBucket()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (b *bs) initCompressor() (err error) {
	// Without compress
	if b.compressAlgorithm == "" {
		return nil
	}

	switch config.CompressAlgorithm(b.compressAlgorithm) {
	case config.GOB:
		b.compressor, err = compress.NewGob()
	case config.GZIP:
		b.compressor, err = compress.NewGzip(
			compress.WithGzipCompressionLevel(b.compressionLevel),
		)
	case config.LZ4:
		b.compressor, err = compress.NewLZ4(
			compress.WithLZ4CompressionLevel(b.compressionLevel),
		)
	case config.ZSTD:
		b.compressor, err = compress.NewZstd(
			compress.WithZstdCompressionLevel(b.compressionLevel),
		)
	default:
		return errors.ErrCompressorNameNotFound(b.compressAlgorithm)
	}

	return err
}

func (b *bs) initBucket() (err error) {
	switch config.AtoBST(b.storageType) {
	case config.S3:
		s, err := session.New(b.s3SessionOpts...).Session()
		if err != nil {
			return err
		}

		b.bucket, err = s3.New(
			append(
				b.s3Opts,
				s3.WithErrGroup(b.eg),
				s3.WithSession(s),
				s3.WithBucket(b.bucketName),
			)...,
		)
		if err != nil {
			return err
		}
	default:
		return errors.ErrInvalidStorageType
	}

	return nil
}

func (b *bs) Reader(ctx context.Context) (r io.ReadCloser, err error) {
	r, err = b.bucket.Reader(ctx, b.filename+b.suffix)
	if err != nil {
		return nil, err
	}

	if b.compressor != nil {
		r, err = b.compressor.Reader(r)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}
