package grp

// GroupExpression can be used to find matching groups
// all fields support "*" as wildcard if they should match everything
type GroupExpression struct {
	// Application
	AppPrefix string
	// name of the cluster
	ClusterName string
	// namespace in the cluster
	Namespace string
	// role in the given context
	Role string
}

//Matches returns if the given groupExpression matches this Group
func (g *GroupExpression) Matches(group Group) bool {

	ok := matchField(group.AppPrefix, g.AppPrefix, false)
	if !ok {
		return false
	}
	ok = matchField(group.ClusterName, g.ClusterName, true)
	if !ok {
		return false
	}
	ok = matchField(group.Namespace, g.Namespace, true)
	if !ok {
		return false
	}
	ok = matchField(group.Role, g.Role, false)
	return ok
}

// matchFiled does a simple match
// supportAll activates that if the value is "all", everything matches
func matchField(value, expression string, supportAll bool) bool {

	if supportAll && value == "all" {
		return true
	}

	if value == expression {
		return true
	}

	if expression == "*" {
		return true
	}

	return false
}
