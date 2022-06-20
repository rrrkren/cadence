// Code generated by "stringer -type=precedence"; DO NOT EDIT.

package ast

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[precedenceUnknown-0]
	_ = x[precedenceTernary-1]
	_ = x[precedenceLogicalOr-2]
	_ = x[precedenceLogicalAnd-3]
	_ = x[precedenceComparison-4]
	_ = x[precedenceNilCoalescing-5]
	_ = x[precedenceBitwiseOr-6]
	_ = x[precedenceBitwiseXor-7]
	_ = x[precedenceBitwiseAnd-8]
	_ = x[precedenceBitwiseShift-9]
	_ = x[precedenceAddition-10]
	_ = x[precedenceMultiplication-11]
	_ = x[precedenceCasting-12]
	_ = x[precedenceUnaryPrefix-13]
	_ = x[precedenceUnaryPostfix-14]
	_ = x[precedenceAccess-15]
	_ = x[precedenceLiteral-16]
}

const _precedence_name = "precedenceUnknownprecedenceTernaryprecedenceLogicalOrprecedenceLogicalAndprecedenceComparisonprecedenceNilCoalescingprecedenceBitwiseOrprecedenceBitwiseXorprecedenceBitwiseAndprecedenceBitwiseShiftprecedenceAdditionprecedenceMultiplicationprecedenceCastingprecedenceUnaryPrefixprecedenceUnaryPostfixprecedenceAccessprecedenceLiteral"

var _precedence_index = [...]uint16{0, 17, 34, 53, 73, 93, 116, 135, 155, 175, 197, 215, 239, 256, 277, 299, 315, 332}

func (i precedence) String() string {
	if i >= precedence(len(_precedence_index)-1) {
		return "precedence(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _precedence_name[_precedence_index[i]:_precedence_index[i+1]]
}
