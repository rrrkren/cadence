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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretTransactions(t *testing.T) {

	t.Parallel()

	t.Run("NoPrepareFunction", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {
              let x = 1 + 2
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("SetTransactionField", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            execute {
              let y = self.x + 1
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("PreConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            pre {
              self.x > 1
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("FailingPreConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            pre {
              self.x > 10
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t,
			ast.ConditionKindPre,
			conditionErr.ConditionKind,
		)
	})

	t.Run("PostConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            execute {
              self.x = 10
            }

            post {
              self.x == 10
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("FailingPostConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            execute {
              self.x = 10
            }

            post {
              self.x == 5
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t,
			ast.ConditionKindPost,
			conditionErr.ConditionKind,
		)
	})

	t.Run("MultipleTransactions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {
              let x = 1 + 2
            }
          }

          transaction {
            execute {
              let y = 3 + 4
            }
          }
        `)

		// first transaction
		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)

		// second transaction
		err = inter.InvokeTransaction(1)
		assert.NoError(t, err)

		// third transaction is not declared
		err = inter.InvokeTransaction(2)
		assert.IsType(t, interpreter.TransactionNotDeclaredError{}, err)
	})

	t.Run("TooFewArguments", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            prepare(signer: &Account) {}
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("TooManyArguments", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {}
          }

          transaction {
            prepare(signer: &Account) {}

            execute {}
          }
        `)

		signer1 := stdlib.NewAccountReferenceValue(nil, nil, interpreter.AddressValue{1}, interpreter.UnauthorizedAccess, interpreter.EmptyLocationRange)
		signer2 := stdlib.NewAccountReferenceValue(nil, nil, interpreter.AddressValue{2}, interpreter.UnauthorizedAccess, interpreter.EmptyLocationRange)

		// first transaction
		err := inter.InvokeTransaction(0, signer1)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)

		// second transaction
		err = inter.InvokeTransaction(0, signer1, signer2)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("Parameters", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let values: [AnyStruct] = []

          transaction(x: Int, y: Bool) {

            prepare(signer: &Account) {
              values.append(signer.address)
              values.append(y)
              values.append(x)
            }
          }
        `)

		arguments := []interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.TrueValue,
		}

		address := common.MustBytesToAddress([]byte{0x1})

		account := stdlib.NewAccountReferenceValue(
			nil,
			nil,
			interpreter.AddressValue(address),
			interpreter.UnauthorizedAccess,
			interpreter.EmptyLocationRange,
		)

		prepareArguments := []interpreter.Value{account}

		arguments = append(arguments, prepareArguments...)

		err := inter.InvokeTransaction(0, arguments...)
		require.NoError(t, err)

		values := inter.Globals.Get("values").GetValue(inter)

		require.IsType(t, &interpreter.ArrayValue{}, values)

		AssertValueSlicesEqual(
			t,
			inter,
			[]interpreter.Value{
				interpreter.AddressValue(address),
				interpreter.TrueValue,
				interpreter.NewUnmeteredIntValueFromInt64(1),
			},
			ArrayElements(inter, values.(*interpreter.ArrayValue)),
		)
	})
}
