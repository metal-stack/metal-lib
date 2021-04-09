package grp

import "strings"

const Any = "*"

// GroupExpression can be used to find matching groups of the schema "[appPrefix]-[firstScope]-[secondScope]-[role]"
// all fields support "*" as wildcard if they should match everything
type GroupExpression struct {
	// Application
	AppPrefix string
	// first resource scope
	FirstScope string
	// second resource scope
	SecondScope string
	// role in the given context
	Role string
}

//Matches returns if the given groupExpression matches this Group
func (g *GroupExpression) Matches(group Group) bool {

	ok := matchField(group.AppPrefix, g.AppPrefix, false)
	if !ok {
		return false
	}
	ok = matchField(group.FirstScope, g.FirstScope, true)
	if !ok {
		return false
	}
	ok = matchField(group.SecondScope, g.SecondScope, true)
	if !ok {
		return false
	}
	ok = matchField(group.Role, g.Role, false)
	return ok
}

// matchFiled does a simple equal-fold-match of the given value with the given expression.
// If expression is "*", every value matches.
// The flag 'supportAll' activates that if the value is "all", everything matches.
func matchField(value, expression string, supportAll bool) bool {

	if supportAll && strings.EqualFold(value, All) {
		return true
	}

	if strings.EqualFold(value, expression) {
		return true
	}

	if expression == Any {
		return true
	}

	return false
}
