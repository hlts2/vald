//
// Copyright (C) 2019 Vdaas.org Vald team ( kpango, kou-m, rinx )
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Package errors provides error types and function
package errors

// "github.com/pkg/errors"

var (

	// Cassandra
	ErrCassandraInvalidConsistencyType = func(consistency string) error {
		return Errorf("consistetncy type %q is not defined", consistency)
	}

	NewErrCassandraNotFound = func(key string) error {
		return &ErrCassandraNotFound{
			err: Errorf("error cassandra key '%s' not found", key),
		}
	}

	ErrCassandraGetOperationFailed = func(key string, err error) error {
		return Wrapf(err, "Failed to fetch key (%s)", key)
	}

	ErrCassandraSetOperationFailed = func(key string, err error) error {
		return Wrapf(err, "Failed to set key (%s)", key)
	}

	ErrCassandraDeleteOperationFailed = func(key string, err error) error {
		return Wrapf(err, "Failed to delete key (%s)", key)
	}
)

type ErrCassandraNotFound struct {
	err error
}

func (e *ErrCassandraNotFound) Error() string {
	return e.err.Error()
}