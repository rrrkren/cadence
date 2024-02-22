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

package capcons

import (
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type CapabilityMigrationReporter interface {
	MigratedPathCapability(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
		borrowType *interpreter.ReferenceStaticType,
	)
	MissingCapabilityID(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
	)
}

// CapabilityValueMigration migrates all path capabilities to ID capabilities,
// using the path to ID capability controller mapping generated by LinkValueMigration.
type CapabilityValueMigration struct {
	CapabilityMapping *CapabilityMapping
	Reporter          CapabilityMigrationReporter
}

var _ migrations.ValueMigration = &CapabilityValueMigration{}

func (*CapabilityValueMigration) Name() string {
	return "CapabilityValueMigration"
}

var fullyEntitledAccountReferenceStaticType = interpreter.ConvertSemaReferenceTypeToStaticReferenceType(
	nil,
	sema.FullyEntitledAccountReferenceType,
)

// Migrate migrates a path capability to an ID capability in the given value.
// If a value is returned, the value must be updated with the replacement in the parent.
// If nil is returned, the value was not updated and no operation has to be performed.
func (m *CapabilityValueMigration) Migrate(
	storageKey interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) (interpreter.Value, error) {
	reporter := m.Reporter

	switch value := value.(type) {
	case *interpreter.PathCapabilityValue: //nolint:staticcheck

		// Migrate the path capability to an ID capability

		oldCapability := value

		capabilityAddressPath := oldCapability.AddressPath()
		capabilityID, controllerBorrowType, ok := m.CapabilityMapping.Get(capabilityAddressPath)
		if !ok {
			if reporter != nil {
				reporter.MissingCapabilityID(
					storageKey.Address,
					capabilityAddressPath,
				)
			}
			break
		}

		oldBorrowType := oldCapability.BorrowType

		// Convert untyped path capability value to typed ID capability value
		// by using capability controller's borrow type
		if oldBorrowType == nil {
			oldBorrowType = interpreter.ConvertSemaToStaticType(nil, controllerBorrowType)
		}

		newBorrowType, ok := oldBorrowType.(*interpreter.ReferenceStaticType)
		if !ok {
			panic(errors.NewUnexpectedError("unexpected non-reference borrow type: %T", oldBorrowType))
		}

		// Convert the old AuthAccount type to the new fully-entitled Account type
		if newBorrowType.ReferencedType == interpreter.PrimitiveStaticTypeAuthAccount { //nolint:staticcheck
			newBorrowType = fullyEntitledAccountReferenceStaticType
		}

		newCapability := interpreter.NewUnmeteredCapabilityValue(
			capabilityID,
			oldCapability.Address,
			newBorrowType,
		)

		if reporter != nil {
			reporter.MigratedPathCapability(
				storageKey.Address,
				capabilityAddressPath,
				newBorrowType,
			)
		}

		return newCapability, nil
	}

	return nil, nil
}
