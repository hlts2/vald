package service

import (
	"archive/tar"
	"context"
	"encoding/gob"
	"io"

	"github.com/vdaas/vald/internal/errgroup"
	ctxio "github.com/vdaas/vald/internal/io"
	"github.com/vdaas/vald/internal/log"
	"github.com/vdaas/vald/internal/safety"
)

const (
	kvsDBName = "ngt-meta.kvsdb"
)

type RebalanceJob interface {
	Start(ctx context.Context) (chan<- error, error)
}

type rebalanceJob struct {
	storage Storage
	eg      errgroup.Group
}

func (rj *rebalanceJob) Start(ctx context.Context) (chan<- error, error) {
	errCh := make(chan error)

	pr, pw := io.Pipe()

	// NOTE: think about error handling

	// 1. Start rebalancer process.
	rj.eg.Go(func() error {
		defer pr.Close()

		// 2. download tar gz file
		rj.eg.Go(safety.RecoverFunc(func() error {
			defer pw.Close()

			sr, err := rj.storage.Reader(ctx)
			if err != nil {
				errCh <- err
				return nil
			}

			sr, err = ctxio.NewReadCloserWithContext(ctx, sr)
			if err != nil {
				errCh <- err
				return nil
			}
			defer func() {
				e := sr.Close()
				if e != nil {
					log.Errorf("error on closing blob-storage reader: %s", e)
				}
			}()

			_, err = io.Copy(pw, sr)
			if err != nil {
				errCh <- err
				return nil
			}

			return nil
		}))

		// 3. unpacka tar file
		// 4. decode vcache filea to get vector ids.
		idm, err := rj.unpackKVSDB(ctx, pr)
		if err != nil {
			errCh <- err
			return nil
		}

		_ = idm

		// 5. calculate to process data from vector ids.

		// 6. send request for getting vector.

		// 7. send request for updateing vector.
		return nil
	})

	return errCh, nil
}

func (rj *rebalanceJob) unpackKVSDB(ctx context.Context, r io.Reader) (map[string]uint32, error) {
	tr := tar.NewReader(r)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if header.Name != kvsDBName {
				continue
			}

			gob.Register(map[string]uint32{})

			// unpacka blob
			idm := make(map[string]uint32)
			err = gob.NewDecoder(tr).Decode(&idm)
			if err != nil {
				return nil, err
			}

			return idm, nil
		}
	}

	// should we return err
	return nil, nil
}
