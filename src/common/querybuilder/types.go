/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package querybuilder

import (
	"fmt"
	"regexp"
	"time"

	"configcenter/src/common"
)

type Rule interface {
	GetDeep() int
	Validate(option *RuleOption) (string, error)
	ToMgo() (mgoFilter map[string]interface{}, errKey string, err error)
	Match(matcher Matcher) bool
}

// *************** define condition ************************
type Condition string

func (c Condition) Validate() error {
	if c == ConditionAnd || c == ConditionOr {
		return nil
	}
	return fmt.Errorf("unexpected condition: %s", c)
}

func (c Condition) ToMgo() (mgoOperator string, err error) {
	switch c {
	case ConditionOr:
		return common.BKDBOR, nil
	case ConditionAnd:
		return common.BKDBAND, nil
	default:
		return "", fmt.Errorf("unexpected operator %s", c)
	}
}

var (
	ConditionAnd = Condition("AND")
	ConditionOr  = Condition("OR")
)

// *************** define operator ************************
type Operator string

var (
	OperatorEqual    = Operator("equal")
	OperatorNotEqual = Operator("not_equal")

	// set operator
	OperatorIn    = Operator("in")
	OperatorNotIn = Operator("not_in")

	// numeric compare
	OperatorLess           = Operator("less")
	OperatorLessOrEqual    = Operator("less_or_equal")
	OperatorGreater        = Operator("greater")
	OperatorGreaterOrEqual = Operator("greater_or_equal")

	// datetime operate
	OperatorDatetimeLess           = Operator("datetime_less")
	OperatorDatetimeLessOrEqual    = Operator("datetime_less_or_equal")
	OperatorDatetimeGreater        = Operator("datetime_greater")
	OperatorDatetimeGreaterOrEqual = Operator("datetime_greater_or_equal")

	// string operator
	OperatorBeginsWith    = Operator("begins_with")
	OperatorNotBeginsWith = Operator("not_begins_with")
	OperatorContains      = Operator("contains")
	OperatorNotContains   = Operator("not_contains")
	OperatorsEndsWith     = Operator("ends_with")
	OperatorNotEndsWith   = Operator("not_ends_with")

	// array operator
	OperatorIsEmpty    = Operator("is_empty")
	OperatorIsNotEmpty = Operator("is_not_empty")

	// null check
	OperatorIsNull    = Operator("is_null")
	OperatorIsNotNull = Operator("is_not_null")

	// exist check
	OperatorExist    = Operator("exist")
	OperatorNotExist = Operator("not_exist")
)

var SupportOperators = map[Operator]bool{
	OperatorEqual:    true,
	OperatorNotEqual: true,

	OperatorIn:    true,
	OperatorNotIn: true,

	OperatorLess:           true,
	OperatorLessOrEqual:    true,
	OperatorGreater:        true,
	OperatorGreaterOrEqual: true,

	OperatorDatetimeLess:           false,
	OperatorDatetimeLessOrEqual:    false,
	OperatorDatetimeGreater:        false,
	OperatorDatetimeGreaterOrEqual: false,

	OperatorBeginsWith:    true,
	OperatorNotBeginsWith: true,
	OperatorContains:      true,
	OperatorNotContains:   true,
	OperatorsEndsWith:     true,
	OperatorNotEndsWith:   true,

	OperatorIsEmpty:    true,
	OperatorIsNotEmpty: true,

	OperatorIsNull:    true,
	OperatorIsNotNull: true,

	OperatorExist:    true,
	OperatorNotExist: true,
}

func (op Operator) Validate() error {
	if support, ok := SupportOperators[op]; !support || !ok {
		return fmt.Errorf("unsupported operator: %s", op)
	}
	return nil
}

// *************** define rule ************************
type AtomRule struct {
	Field    string      `json:"field"`
	Operator Operator    `json:"operator"`
	Value    interface{} `json:"value"`
}

func (r AtomRule) GetDeep() int {
	return int(1)
}

func (r AtomRule) Validate(option *RuleOption) (string, error) {
	if err := r.Operator.Validate(); err != nil {
		return "operator", err
	}
	if err := r.validateField(); err != nil {
		return "field", err
	}
	if err := r.validateValue(option); err != nil {
		return "value", err
	}
	return "", nil
}

type Matcher func(r AtomRule) bool

func (r AtomRule) Match(matcher Matcher) bool {
	return matcher(r)
}

var (
	// TODO: should we support dot field separator here?
	ValidFieldPattern = regexp.MustCompile(`^[a-zA-Z0-9_][\d\w\-_.]*$`)
)

func (r AtomRule) validateField() error {
	if !ValidFieldPattern.MatchString(r.Field) {
		return fmt.Errorf("invalid field: %s", r.Field)
	}
	return nil
}

func (r AtomRule) validateValue(option *RuleOption) error {
	switch r.Operator {
	case OperatorEqual, OperatorNotEqual:
		return validateBasicType(r.Value)

	case OperatorIn, OperatorNotIn:
		return validateSliceOfBasicType(r.Value, option.NeedSameSliceElementType, option.MaxSliceElementsCount)

	case OperatorLess, OperatorLessOrEqual, OperatorGreater, OperatorGreaterOrEqual:
		return validateNumericType(r.Value)
	case OperatorDatetimeLess, OperatorDatetimeLessOrEqual, OperatorDatetimeGreater, OperatorDatetimeGreaterOrEqual:
		return validateDatetimeStringType(r.Value)
	case OperatorBeginsWith, OperatorNotBeginsWith, OperatorContains, OperatorNotContains, OperatorsEndsWith, OperatorNotEndsWith:
		return validateNotEmptyStringType(r.Value)
	case OperatorIsEmpty, OperatorIsNotEmpty:
		return nil
	case OperatorIsNull, OperatorIsNotNull:
		return nil
	case OperatorExist, OperatorNotExist:
		return nil
	default:
		return fmt.Errorf("unsupported operator: %s", r.Operator)
	}
}

// ToMgo generate mongo filter from rule
func (r AtomRule) ToMgo() (mgoFiler map[string]interface{}, key string, err error) {
	if key, err := r.Validate(&RuleOption{NeedSameSliceElementType: true}); err != nil {
		return nil, key, fmt.Errorf("validate failed, key: %s, err: %s", key, err)
	}

	filter := make(map[string]interface{})
	switch r.Operator {
	case OperatorEqual:
		filter[r.Field] = map[string]interface{}{
			common.BKDBEQ: r.Value,
		}
	case OperatorNotEqual:
		filter[r.Field] = map[string]interface{}{
			common.BKDBNE: r.Value,
		}
	case OperatorIn:
		filter[r.Field] = map[string]interface{}{
			common.BKDBIN: r.Value,
		}
	case OperatorNotIn:
		filter[r.Field] = map[string]interface{}{
			common.BKDBNIN: r.Value,
		}
	case OperatorLess:
		filter[r.Field] = map[string]interface{}{
			common.BKDBLT: r.Value,
		}
	case OperatorLessOrEqual:
		filter[r.Field] = map[string]interface{}{
			common.BKDBLTE: r.Value,
		}
	case OperatorGreater:
		filter[r.Field] = map[string]interface{}{
			common.BKDBGT: r.Value,
		}
	case OperatorGreaterOrEqual:
		filter[r.Field] = map[string]interface{}{
			common.BKDBGTE: r.Value,
		}
	case OperatorDatetimeLess:
		t, err := time.Parse(time.RFC3339, r.Value.(string))
		if err != nil {
			return nil, "value", err
		}
		filter[r.Field] = map[string]interface{}{
			common.BKDBLT: t,
		}
	case OperatorDatetimeLessOrEqual:
		t, err := time.Parse(time.RFC3339, r.Value.(string))
		if err != nil {
			return nil, "value", err
		}
		filter[r.Field] = map[string]interface{}{
			common.BKDBLTE: t,
		}
	case OperatorDatetimeGreater:
		t, err := time.Parse(time.RFC3339, r.Value.(string))
		if err != nil {
			return nil, "value", err
		}
		filter[r.Field] = map[string]interface{}{
			common.BKDBGT: t,
		}
	case OperatorDatetimeGreaterOrEqual:
		t, err := time.Parse(time.RFC3339, r.Value.(string))
		if err != nil {
			return nil, "value", err
		}
		filter[r.Field] = map[string]interface{}{
			common.BKDBGTE: t,
		}
	case OperatorBeginsWith:
		filter[r.Field] = map[string]interface{}{
			common.BKDBLIKE: fmt.Sprintf("^%s", r.Value),
		}
	case OperatorNotBeginsWith:
		filter[r.Field] = map[string]interface{}{
			common.BKDBNot: map[string]interface{}{common.BKDBLIKE: fmt.Sprintf("^%s", r.Value)},
		}
	case OperatorContains:
		filter[r.Field] = map[string]interface{}{
			common.BKDBLIKE:    fmt.Sprintf("%s", r.Value),
			common.BKDBOPTIONS: "i",
		}
	case OperatorNotContains:
		filter[r.Field] = map[string]interface{}{
			common.BKDBNot: map[string]interface{}{common.BKDBLIKE: fmt.Sprintf("%s", r.Value)},
		}
	case OperatorsEndsWith:
		filter[r.Field] = map[string]interface{}{
			common.BKDBLIKE: fmt.Sprintf("%s$", r.Value),
		}
	case OperatorNotEndsWith:
		filter[r.Field] = map[string]interface{}{
			common.BKDBNot: map[string]interface{}{common.BKDBLIKE: fmt.Sprintf("%s$", r.Value)},
		}
	case OperatorIsEmpty:
		// array empty
		filter[r.Field] = map[string]interface{}{
			common.BKDBEQ: make([]interface{}, 0),
		}
	case OperatorIsNotEmpty:
		// array not empty
		filter[r.Field] = map[string]interface{}{
			common.BKDBNE: make([]interface{}, 0),
		}
	case OperatorIsNull:
		filter[r.Field] = map[string]interface{}{
			common.BKDBEQ: nil,
		}
	case OperatorIsNotNull:
		filter[r.Field] = map[string]interface{}{
			common.BKDBNE: nil,
		}
	case OperatorExist:
		filter[r.Field] = map[string]interface{}{
			common.BKDBExists: true,
		}
	case OperatorNotExist:
		filter[r.Field] = map[string]interface{}{
			common.BKDBExists: false,
		}
	default:
		return nil, "operator", fmt.Errorf("unsupported operator: %s", r.Operator)
	}
	return filter, "", nil
}

// *************** define query ************************
type CombinedRule struct {
	Condition Condition `json:"condition"`
	Rules     []Rule    `json:"rules"`
}

var (
	// 嵌套层级的深度按树的高度计算，查询条件最大深度为3即最多嵌套2层
	MaxDeep = 3

	// DefaultMaxSliceElementsCount is max elements count of slice(array) condition value.
	DefaultMaxSliceElementsCount = 500

	// DefaultMaxConditionOrRulesCount is default max rules count of one OR combined condition.
	DefaultMaxConditionOrRulesCount = 20
)

// RuleOption is combined condition rule validator option.
type RuleOption struct {
	// NeedSameSliceElementType whether need same type in one slice(array) value or not.
	NeedSameSliceElementType bool

	// MaxSliceElementsCount max slice(array) value elements count, 0 means no limit.
	MaxSliceElementsCount int

	// MaxConditionOrRulesCount max atom rules count in one OR combined condition, 0 means no limit.
	MaxConditionOrRulesCount int

	// MaxConditionAndRulesCount max atom rules count in one AND combined condition, 0 means no limit.
	MaxConditionAndRulesCount int
}

func (r CombinedRule) GetDeep() int {
	maxChildDeep := 1
	for _, child := range r.Rules {
		childDeep := child.GetDeep()
		if childDeep > maxChildDeep {
			maxChildDeep = childDeep
		}
	}
	return maxChildDeep + 1
}

// Validate validates combined rules with the options.
func (r CombinedRule) Validate(option *RuleOption) (string, error) {
	if err := r.Condition.Validate(); err != nil {
		return "condition", err
	}

	if len(r.Rules) == 0 {
		return "rules", fmt.Errorf("combined rules shouldn't be empty")
	}

	// validate condition rules count.
	if r.Condition == ConditionOr && option.MaxConditionOrRulesCount > 0 &&
		len(r.Rules) > option.MaxConditionOrRulesCount {

		return "rules", fmt.Errorf("too many rules of OR condition: %d max(%d)",
			len(r.Rules), option.MaxConditionOrRulesCount)
	}

	if r.Condition == ConditionAnd && option.MaxConditionAndRulesCount > 0 &&
		len(r.Rules) > option.MaxConditionAndRulesCount {

		return "rules", fmt.Errorf("too many rules of AND condition: %d max(%d)",
			len(r.Rules), option.MaxConditionAndRulesCount)
	}

	for idx, rule := range r.Rules {
		if key, err := rule.Validate(option); err != nil {
			return fmt.Sprintf("rules[%d].%s", idx, key), err
		}
	}
	return "", nil
}

func (r CombinedRule) ToMgo() (mgoFilter map[string]interface{}, key string, err error) {
	if err := r.Condition.Validate(); err != nil {
		return nil, "condition", err
	}
	if r.Rules == nil || len(r.Rules) == 0 {
		return nil, "rules", fmt.Errorf("combined rules shouldn't be empty")
	}
	filters := make([]map[string]interface{}, 0)
	for idx, rule := range r.Rules {
		filter, key, err := rule.ToMgo()
		if err != nil {
			return nil, fmt.Sprintf("rules[%d].%s", idx, key), err
		}
		filters = append(filters, filter)
	}
	mgoOperator, err := r.Condition.ToMgo()
	if err != nil {
		return nil, "condition", err
	}
	mgoFilter = map[string]interface{}{
		mgoOperator: filters,
	}
	return mgoFilter, "", nil
}

func (r CombinedRule) Match(matcher Matcher) bool {
	if len(r.Rules) == 0 {
		return true
	}

	switch r.Condition {
	case ConditionAnd:
		for _, rule := range r.Rules {
			if rule.Match(matcher) == false {
				return false
			}
		}
		return true
	case ConditionOr:
		for _, rule := range r.Rules {
			if rule.Match(matcher) == true {
				return true
			}
		}
		return false
	default:
		panic(fmt.Sprintf("unexpected condition %s", r.Condition))
	}
}
