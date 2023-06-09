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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckOptionalChainingNonOptionalFieldRead(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Test {
          let x: Int

          init(x: Int) {
              self.x = x
          }
      }

      let test: Test? = Test(x: 1)
      let x = test?.x
    `)

	require.NoError(t, err)

	assert.Equal(t,
		&sema.OptionalType{Type: sema.IntType},
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckOptionalChainingOptionalFieldRead(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Test {
          let x: Int?

          init(x: Int?) {
              self.x = x
          }
      }

      let test: Test? = Test(x: 1)
      let x = test?.x
    `)

	require.NoError(t, err)

	assert.Equal(t,
		&sema.OptionalType{Type: sema.IntType},
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}

func TestCheckOptionalChainingNonOptionalFieldAccess(t *testing.T) {

	t.Parallel()

	t.Run("function", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
              fun test() {
                  let bar = Bar()
                  // field Bar.foo is not optional but try to access it through optional chaining
                  bar.foo?.getContent()
              }

              struct Bar {
                  var foo: Foo
                  init() {
                      self.foo = Foo()
                  }
              }

              struct Foo {
                  fun getContent(): String {
                      return "hello"
                  }
              }
            `,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidOptionalChainingError{}, errs[0])

	})

	t.Run("non-function", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
              fun test() {
                  let bar = Bar()
                  // Two issues:
                  //    - Field Bar.foo is not optional, but access through optional chaining
                  //    - Field Foo.id is not a function, yet invoke as a function
                  bar.foo?.id()
              }

              struct Bar {
                  var foo: Foo
                  init() {
                      self.foo = Foo()
                  }
              }

              struct Foo {
                  var id: String

                  init() {
                      self.id = ""
                  }
              }
            `,
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidOptionalChainingError{}, errs[0])
		assert.IsType(t, &sema.NotCallableError{}, errs[1])
	})
}

func TestCheckOptionalChainingFunctionRead(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Test {
          fun x(): Int {
              return 42
          }
      }

      let test: Test? = Test()
      let x = test?.x
    `)

	require.NoError(t, err)

	xType := RequireGlobalValue(t, checker.Elaboration, "x")

	expectedType := &sema.OptionalType{
		Type: &sema.FunctionType{
			Purity:               sema.FunctionPurityImpure,
			ReturnTypeAnnotation: sema.IntTypeAnnotation,
		},
	}

	assert.True(t, xType.Equal(expectedType))
}

func TestCheckOptionalChainingFunctionCall(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Test {
          fun x(): Int {
              return 42
          }
      }

      let test: Test? = Test()
      let x = test?.x()
    `)

	require.NoError(t, err)

	assert.True(t,
		RequireGlobalValue(t, checker.Elaboration, "x").Equal(
			&sema.OptionalType{Type: sema.IntType},
		),
	)
}

func TestCheckInvalidOptionalChainingNonOptional(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          let x: Int

          init(x: Int) {
              self.x = x
          }
      }

      let test = Test(x: 1)
      let x = test?.x
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidOptionalChainingError{}, errs[0])
}

func TestCheckInvalidOptionalChainingFieldAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct Test {
          var x: Int
          init(x: Int) {
              self.x = x
          }
      }

      fun test() {
          let test: Test? = Test(x: 1)
          test?.x = 2
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UnsupportedOptionalChainingAssignmentError{}, errs[0])
}

func TestCheckFunctionTypeReceiverType(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          struct S {
              fun f() {}
          }

          let s = S()
          let f = s.f
        `)

		require.NoError(t, err)

		assert.Equal(t,
			&sema.FunctionType{
				Purity:               sema.FunctionPurityImpure,
				Parameters:           []sema.Parameter{},
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
			RequireGlobalValue(t, checker.Elaboration, "f"),
		)
	})

	t.Run("cast bound function type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              fun f() {}
          }

          let s = S()
          let f = s.f as fun(): Void
        `)

		require.NoError(t, err)
	})
}

func TestCheckMemberNotDeclaredSecondaryError(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t, `
            struct Test {
                fun foo(): Int { return 3 }
            }

            let test: Test = Test()
            let x = test.foop()
        `, ParseAndCheckOptions{
			Config: &sema.Config{
				SuggestionsEnabled: true,
			},
		})

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "did you mean `foo`?", memberErr.SecondaryError())
	})

	t.Run("without option", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Test {
                fun foo(): Int { return 3 }
            }

            let test: Test = Test()
            let x = test.foop()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "unknown member", memberErr.SecondaryError())
	})

	t.Run("selects closest", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t, `
            struct Test {
                fun fou(): Int { return 1 }
                fun bar(): Int { return 2 }
                fun foo(): Int { return 3 }
            }

            let test: Test = Test()
            let x = test.foop()
        `, ParseAndCheckOptions{
			Config: &sema.Config{
				SuggestionsEnabled: true,
			},
		})

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "did you mean `foo`?", memberErr.SecondaryError())
	})

	t.Run("no members = no suggestion", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t, `
            struct Test {
                
            }

            let test: Test = Test()
            let x = test.foop()
        `, ParseAndCheckOptions{
			Config: &sema.Config{
				SuggestionsEnabled: true,
			},
		})

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "unknown member", memberErr.SecondaryError())
	})

	t.Run("no similarity = no suggestion", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t, `
            struct Test {
                fun bar(): Int { return 1 }
            }

            let test: Test = Test()
            let x = test.foop()
        `, ParseAndCheckOptions{
			Config: &sema.Config{
				SuggestionsEnabled: true,
			},
		})

		errs := RequireCheckerErrors(t, err, 1)

		var memberErr *sema.NotDeclaredMemberError
		require.ErrorAs(t, errs[0], &memberErr)
		assert.Equal(t, "unknown member", memberErr.SecondaryError())
	})
}

func TestCheckMemberAccess(t *testing.T) {

	t.Parallel()

	t.Run("composite, field", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            struct Test {
                var x: [Int]
                init() {
                    self.x = []
                }
            }

            fun test() {
                let test = Test()
                var x: [Int] = test.x
            }
        `)

		require.NoError(t, err)
	})

	t.Run("composite, function", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            struct Test {
                pub fun foo(): Int {
                    return 1
                }
            }

            fun test() {
                let test = Test()
                var foo: (fun(): Int) = test.foo
            }
        `)

		require.NoError(t, err)
	})

	t.Run("composite reference, field", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            struct Test {
                var x: [Int]
                init() {
                    self.x = []
                }
            }

            fun test() {
                let test = Test()
                let testRef = &test as &Test
                var x: &[Int] = testRef.x
            }
        `)

		require.NoError(t, err)
	})

	t.Run("composite reference, optional field", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            struct Test {
                var x: [Int]?
                init() {
                    self.x = []
                }
            }

            fun test() {
                let test = Test()
                let testRef = &test as &Test
                var x: &[Int]? = testRef.x
            }
        `)

		require.NoError(t, err)
	})

	t.Run("composite reference, primitive field", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            struct Test {
                var x: Int
                init() {
                    self.x = 1
                }
            }

            fun test() {
                let test = Test()
                let testRef = &test as &Test
                var x: Int = testRef.x
            }
        `)

		require.NoError(t, err)
	})

	t.Run("composite reference, function", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            struct Test {
                pub fun foo(): Int {
                    return 1
                }
            }

            fun test() {
                let test = Test()
                let testRef = &test as &Test
                var foo: (fun(): Int) = testRef.foo
            }
        `)

		require.NoError(t, err)
	})

	t.Run("array, element", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
                let array: [[Int]] = [[1, 2]]
                var x: [Int] = array[0]
            }
        `)

		require.NoError(t, err)
	})

	t.Run("array reference, element", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
                let array: [[Int]] = [[1, 2]]
                let arrayRef = &array as &[[Int]]
                var x: &[Int] = arrayRef[0]
            }
        `)

		require.NoError(t, err)
	})

	t.Run("array reference, optional typed element", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
                let array: [[Int]?] = [[1, 2]]
                let arrayRef = &array as &[[Int]?]
                var x: &[Int]? = arrayRef[0]
            }
        `)

		require.NoError(t, err)
	})

	t.Run("array reference, primitive typed element", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
                let array: [Int] = [1, 2]
                let arrayRef = &array as &[Int]
                var x: Int = arrayRef[0]
            }
        `)

		require.NoError(t, err)
	})

	t.Run("dictionary, value", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
                let dict: {String: {String: Int}} = {"one": {"two": 2}}
                var x: {String: Int}? = dict["one"]
            }
        `)

		require.NoError(t, err)
	})

	t.Run("dictionary reference, value", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
                let dict: {String: {String: Int} } = {"one": {"two": 2}}
                let dictRef = &dict as &{String: {String: Int}}
                var x: &{String: Int}? = dictRef["one"]
            }
        `)

		require.NoError(t, err)
	})

	t.Run("dictionary reference, optional typed value", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
                let dict: {String: {String: Int}?} = {"one": {"two": 2}}
                let dictRef = &dict as &{String: {String: Int}?}
                var x: (&{String: Int})?? = dictRef["one"]
            }
        `)

		require.NoError(t, err)
	})

	t.Run("dictionary reference, optional typed value, mismatch types", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
                let dict: {String: {String: Int}?} = {"one": {"two": 2}}
                let dictRef = &dict as &{String: {String: Int}?}

                // Must return an optional reference, not a reference to an optional
                var x: &({String: Int}??) = dictRef["one"]
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)
		typeMismatchError := &sema.TypeMismatchError{}
		require.ErrorAs(t, errors[0], &typeMismatchError)
	})

	t.Run("dictionary reference, primitive typed value", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
                let dict: {String: Int} = {"one": 1}
                let dictRef = &dict as &{String: Int}
                var x: Int? = dictRef["one"]
            }
        `)

		require.NoError(t, err)
	})

	t.Run("resource reference, attachment", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            resource R {}

            attachment A for R {}

            fun test() {
                let r <- create R()
                let rRef = &r as &R

                var a: &A? = rRef[A]
                destroy r
            }
        `)

		require.NoError(t, err)
	})
}
