/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package migrations

import (
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

type AccountStorage struct {
	storage *runtime.Storage
	address common.Address
}

// NewAccountStorage constructs an `AccountStorage` for a given account.
func NewAccountStorage(storage *runtime.Storage, address common.Address) AccountStorage {
	return AccountStorage{
		storage: storage,
		address: address,
	}
}

// ForEachValue iterates over the values in the account.
func (i *AccountStorage) ForEachValue(
	inter *interpreter.Interpreter,
	domains []common.PathDomain,
	valueConverter func(interpreter.Value) (newValue interpreter.Value, updated bool),
	reporter Reporter,
) {
	for _, domain := range domains {
		storageMap := i.storage.GetStorageMap(i.address, domain.Identifier(), false)
		if storageMap == nil || storageMap.Count() == 0 {
			continue
		}

		iterator := storageMap.Iterator(inter)

		for key, value := iterator.Next(); key != nil; key, value = iterator.Next() {
			newValue, updated := valueConverter(value)
			if newValue == nil && !updated {
				continue
			}

			identifier := string(key.(interpreter.StringAtreeValue))

			if newValue != nil {
				// If the converter returns a new value, then replace the existing value with the new one.
				storageMap.SetValue(
					inter,
					interpreter.StringStorageMapKey(identifier),
					newValue,
				)
			}

			reporter.Report(i.address, domain, identifier)
		}
	}
}
